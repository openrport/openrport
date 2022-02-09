package chclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cloudradar-monitoring/rport/client/system"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
)

// now is used to stub time.Now in tests
var now = time.Now

func (c *Client) HandleRunCmdRequest(ctx context.Context, reqPayload []byte) (*comm.RunCmdResponse, error) {
	if !c.configHolder.RemoteCommands.Enabled {
		return nil, errors.New("remote commands execution is disabled")
	}

	job := models.Job{}
	err := json.Unmarshal(reqPayload, &job)
	if err != nil {
		return nil, fmt.Errorf("failed to decode requested job: %s", err)
	}

	if job.IsScript && !c.configHolder.RemoteScripts.Enabled {
		return nil, errors.New("remote scripts are disabled")
	}

	if !job.IsScript && !c.isAllowed(job.Command) {
		return nil, fmt.Errorf("command is not allowed: %v", job.Command)
	}

	interpreter := system.Interpreter{
		InterpreterNameFromInput: job.Interpreter,
		InterpreterAliases:       c.configHolder.InterpreterAliases,
	}

	scriptPath, err := system.CreateScriptFile(c.configHolder.GetScriptsDir(), job.Command, interpreter)
	if err != nil {
		return nil, err
	}

	execCtx := &system.CmdExecutorContext{
		Interpreter: interpreter,
		Command:     scriptPath,
		WorkingDir:  job.Cwd,
		IsSudo:      job.IsSudo,
		HasShebang:  system.HasShebangLine(job.Command),
	}
	cmd := c.cmdExec.New(ctx, execCtx)
	stdOut := &CapacityBuffer{capacity: c.configHolder.RemoteCommands.SendBackLimit}
	stdErr := &CapacityBuffer{capacity: c.configHolder.RemoteCommands.SendBackLimit}
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr

	c.Debugf("Input command: %s, sysProcAttributes: %+v, executable command: %s", job.Command, cmd.SysProcAttr, cmd.String())

	startedAt := now()
	err = c.cmdExec.Start(cmd)
	if err != nil {
		c.rmScript(scriptPath)
		return nil, fmt.Errorf("failed to start a command: %s", err)
	}

	res := &comm.RunCmdResponse{
		Pid:       cmd.Process.Pid,
		StartedAt: startedAt,
	}

	// observe the cmd execution in background
	go func() {
		defer c.rmScript(scriptPath)

		c.Debugf("started to observe cmd [jid=%q,pid=%d]", job.JID, res.Pid)

		// after timeout stop observing but leave the cmd running
		done := make(chan error)
		go func() { done <- c.cmdExec.Wait(cmd) }()

		var status string
		var execErr error
		select {
		case execErr = <-done:
			if execErr != nil {
				status = models.JobStatusFailed
				c.Errorf("failed to run command[jid=%q,pid=%d]:\ncmd:\n%s\nerr: %s", job.JID, res.Pid, job.Command, execErr)
			} else {
				status = models.JobStatusSuccessful
			}
		case <-time.After(time.Duration(job.TimeoutSec) * time.Second):
			status = models.JobStatusUnknown
			c.Debugf("timeout (%d seconds) reached, stop observing command[jid=%q,pid=%d]:\n%s", job.TimeoutSec, job.JID, res.Pid, job.Command)
		}

		// fill all unset fields
		now := now()
		job.FinishedAt = &now
		job.Status = status
		job.PID = &res.Pid
		job.StartedAt = startedAt

		job.Error = c.buildErrText(execErr, stdOut, stdErr)
		if job.Error != "" {
			c.Errorf(job.Error)
		}

		job.Result = &models.JobResult{
			StdOut: c.ToUTF8(stdOut.Bytes()),
			StdErr: c.ToUTF8(stdErr.Bytes()),
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

func (c *Client) buildErrText(execErr error, stdOut, stdErr *CapacityBuffer) string {
	errs := make([]string, 0, 3)

	if execErr != nil {
		errs = append(errs, execErr.Error())
	}
	if stdOut.HasOverflow() {
		errs = append(errs, fmt.Sprintf("overflow of stdOut buffer: %s", stdOut.GetOverflowMessage()))
	}
	if stdErr.HasOverflow() {
		errs = append(errs, fmt.Sprintf("overflow of stdErr buffer: %s", stdErr.GetOverflowMessage()))
	}

	return strings.Join(errs, ", ")
}

func (c *Client) rmScript(scriptPath string) {
	err := os.Remove(scriptPath)
	if err != nil {
		c.Errorf("failed to delete script %s: %v", scriptPath, err)
	} else {
		c.Debugf("deleted script %s after execution", scriptPath)
	}
}

// isAllowed returns true if a given command passes configured restrictions.
func (c *Client) isAllowed(cmd string) bool {
	allowMatch := matchRegexp(cmd, c.configHolder.RemoteCommands.AllowRegexp)
	denyMatch := matchRegexp(cmd, c.configHolder.RemoteCommands.DenyRegexp)
	switch c.configHolder.RemoteCommands.Order {
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

func (c *Client) ToUTF8(b []byte) string {
	if c.consoleDecoder == nil {
		return string(b)
	}

	decoded, err := c.consoleDecoder.Bytes(b)
	if err != nil {
		// just log and return original
		c.Infof("could not convert cmd output to UTF-8: %v", err)
		return string(b)
	}

	return string(decoded)
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
	data        []byte
	capacity    int
	hasOverflow bool
}

func (b *CapacityBuffer) HasOverflow() bool {
	return b.hasOverflow
}

func (b *CapacityBuffer) GetOverflowMessage() string {
	return fmt.Sprintf("maximum send_back_limit of %d bytes exceeded", b.capacity)
}

func (b *CapacityBuffer) Write(p []byte) (n int, err error) {
	freeCapacity := b.capacity - len(b.data)

	// do not write to buffer if no space left
	if len(p) > freeCapacity {
		b.data = append(b.data, p[:freeCapacity]...)
		b.hasOverflow = true
		return 0, errors.New(b.GetOverflowMessage())
	}

	b.data = append(b.data, p...)

	return len(p), nil
}

func (b *CapacityBuffer) String() string {
	return string(b.data)
}

func (b *CapacityBuffer) Bytes() []byte {
	return b.data
}
