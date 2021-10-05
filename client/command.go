package chclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"time"

	"github.com/cloudradar-monitoring/rport/client/system"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func (c *Client) HandleRunCmdRequest(ctx context.Context, reqPayload []byte) (*comm.RunCmdResponse, error) {
	if !c.config.RemoteCommands.Enabled {
		return nil, errors.New("remote commands execution is disabled")
	}

	job := models.Job{}
	err := json.Unmarshal(reqPayload, &job)
	if err != nil {
		return nil, fmt.Errorf("failed to decode requested job: %s", err)
	}

	// do not accept a new request when the previous is not finished yet, except multi-client job. In this case wait
	// NOTE: HandleRunCmdRequest is run sequentially, that's why no need to lock a block with read/write curPID
	curPID := c.getCurCmdPID()
	if curPID != nil {
		if job.MultiJobID == nil {
			return nil, fmt.Errorf("a previous command execution with PID %d is still running", *curPID)
		}
		c.Debugf("Waiting for a previous command with PID %d to finish", *curPID)
	}

	if job.IsScript && !c.config.RemoteScripts.Enabled {
		return nil, errors.New("remote scripts are disabled")
	}

	// TODO: temporary solution, refactor with using worker pool
	c.runCmdMutex.Lock()

	job.Interpreter, err = system.GetInterpreter(job.Interpreter, runtime.GOOS, job.HasShebang)
	if err != nil {
		c.runCmdMutex.Unlock()
		return nil, err
	}

	if !c.isAllowed(job.Command) {
		c.runCmdMutex.Unlock()
		return nil, fmt.Errorf("command is not allowed: %v", job.Command)
	}

	execCtx := &system.CmdExecutorContext{
		Interpreter: job.Interpreter,
		Command:     job.Command,
		WorkingDir:  job.Cwd,
		IsSudo:      job.IsSudo,
		IsScript:    job.IsScript,
	}
	cmd := c.cmdExec.New(ctx, execCtx)
	stdOut := CapacityBuffer{capacity: c.config.RemoteCommands.SendBackLimit}
	stdErr := CapacityBuffer{capacity: c.config.RemoteCommands.SendBackLimit}
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	c.Debugf("Generated command is %s, sysProcAttributes: %+v", cmd.String(), cmd.SysProcAttr)

	startedAt := system.Now()
	err = c.cmdExec.Start(cmd)
	if err != nil {
		c.runCmdMutex.Unlock()
		c.rmScriptIfNeeded(job.Command, job.IsScript)
		return nil, fmt.Errorf("failed to start a command: %s", err)
	}

	// set running PID
	c.setCurCmdPID(&cmd.Process.Pid)

	res := &comm.RunCmdResponse{
		Pid:       cmd.Process.Pid,
		StartedAt: startedAt,
	}

	// observe the cmd execution in background
	go func() {
		defer c.rmScriptIfNeeded(job.Command, job.IsScript)

		c.Debugf("started to observe cmd [jid=%q,pid=%d]", job.JID, res.Pid)

		// after timeout stop observing but leave the cmd running
		done := make(chan error)
		go func() { done <- c.cmdExec.Wait(cmd) }()

		var status string
		select {
		case err := <-done:
			if err != nil {
				status = models.JobStatusFailed
				c.Errorf("failed to run command[jid=%q,pid=%d]:\ncmd:\n%s\nerr: %s", job.JID, res.Pid, job.Command, err)
			} else {
				status = models.JobStatusSuccessful
			}
		case <-time.After(time.Duration(job.TimeoutSec) * time.Second):
			status = models.JobStatusUnknown
			c.Debugf("timeout (%d seconds) reached, stop observing command[jid=%q,pid=%d]:\n%s", job.TimeoutSec, job.JID, res.Pid, job.Command)
		}

		// observing stopped - unset PID
		c.setCurCmdPID(nil)
		c.runCmdMutex.Unlock()

		// fill all unset fields
		now := system.Now()
		job.FinishedAt = &now
		job.Status = status
		job.PID = &res.Pid
		job.StartedAt = startedAt

		job.Result = &models.JobResult{
			StdOut: stdOut.String(),
			StdErr: stdErr.String(),
		}

		// send the filled job to the server
		jobBytes, err := json.Marshal(job)
		if err != nil {
			c.Errorf("failed to send command result for [jid=%q,pid=%d]: failed to encode job result: %s", job.JID, res.Pid, err)
			return
		}
		c.Debugf("sending job to server: %v", job)
		_, _, err = c.sshConn.SendRequest(comm.RequestTypeCmdResult, false, jobBytes)
		if err != nil {
			c.Errorf("failed to send command result to server[jid=%q,pid=%d]: %s", job.JID, res.Pid, err)
		}

		c.Debugf("finished to observe cmd [jid=%q,pid=%d]", job.JID, res.Pid)
	}()

	return res, nil
}

func (c *Client) rmScriptIfNeeded(scriptPath string, isScript bool) {
	if !isScript {
		return
	}

	err := os.Remove(scriptPath)
	if err != nil {
		c.Errorf("failed to delete script %s: %v", scriptPath, err)
	} else {
		c.Debugf("deleted script %s after execution", scriptPath)
	}
}

// isAllowed returns true if a given command passes configured restrictions.
func (c *Client) isAllowed(cmd string) bool {
	allowMatch := matchRegexp(cmd, c.config.RemoteCommands.allowRegexp)
	denyMatch := matchRegexp(cmd, c.config.RemoteCommands.denyRegexp)
	switch c.config.RemoteCommands.Order {
	case allowDenyOrder:
		if !allowMatch {
			return false
		}
		return !denyMatch
	case denyAllowOrder:
		if allowMatch {
			return true
		}
		return !denyMatch
	}
	return false
}

// matchRegexp returns true if a given command matches at least one of given regular expressions.
func matchRegexp(cmd string, regexpList []*regexp.Regexp) bool {
	for _, regx := range regexpList {
		if regx.MatchString(cmd) {
			return true
		}
	}
	return false
}

type CapacityBuffer struct {
	bytes.Buffer
	capacity int
}

func (b *CapacityBuffer) Write(p []byte) (n int, err error) {
	freeCapacity := b.capacity - b.Len()

	// do not write to buffer if no space left
	if freeCapacity <= 0 {
		return len(p), nil // pretend a successful write
	}

	// write to buffer only a part if exceeds the capacity
	if len(p) > freeCapacity {
		return b.Buffer.Write(p[:freeCapacity])
	}

	return b.Buffer.Write(p)
}
