package chclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"time"

	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type CmdExecutorContext struct {
	Shell      string
	Command    string
	WorkingDir string
	IsSudo     bool
	IsScript   bool
}

type CmdExecutor interface {
	New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd
	Start(cmd *exec.Cmd) error
	Wait(cmd *exec.Cmd) error
}

type CmdExecutorImpl struct {
}

func NewCmdExecutor() *CmdExecutorImpl {
	return &CmdExecutorImpl{}
}

func (e *CmdExecutorImpl) Start(cmd *exec.Cmd) error {
	return cmd.Start()
}

func (e *CmdExecutorImpl) Wait(cmd *exec.Cmd) error {
	return cmd.Wait()
}

func (e *CmdExecutorImpl) newCmd(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	var args []string
	if execCtx.IsSudo {
		args = append(args, "sudo -n")
	}

	args = append(args, shellOptions[execCtx.Shell]...)

	additionalArgs := getAdditionalArgs(execCtx.IsScript, execCtx.Shell)
	args = append(args, additionalArgs...)

	args = append(args, execCtx.Command)
	cmd := exec.CommandContext(ctx, execCtx.Shell, args...)
	cmd.Dir = execCtx.WorkingDir
	return cmd
}

func getAdditionalArgs(isScript bool, shell string) []string {
	if shell == "" {
		return []string{}
	}

	if isScript {
		scriptOptions, ok := shellOptionsScript[shell]
		if ok {
			return scriptOptions
		}
		return []string{}
	}

	commandOptions, ok := shellOptionsCommand[shell]
	if ok {
		return commandOptions
	}

	return []string{}
}

const (
	unixShell  = "/bin/sh"
	cmdShell   = "cmd"
	powerShell = "powershell"
)

var shellOptions = map[string][]string{
	// in order to run multiple commands under one process run with these options
	unixShell: {"-c"},
	cmdShell:  {"/c"},
	powerShell: {
		"-Noninteractive", // Don't present an interactive prompt to the user.
		"-executionpolicy",
		"bypass",
	},
}

var shellOptionsCommand = map[string][]string{
	powerShell: {
		"-Command",
	},
}

var shellOptionsScript = map[string][]string{
	powerShell: {
		"-File",
	},
}

// now is used to stub time.Now in tests
var now = time.Now

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

	// TODO: temporary solution, refactor with using worker pool
	c.runCmdMutex.Lock()

	job.Shell, err = getShell(job.Shell, runtime.GOOS)
	if err != nil {
		c.runCmdMutex.Unlock()
		return nil, err
	}

	if !c.isAllowed(job.Command) {
		c.runCmdMutex.Unlock()
		return nil, fmt.Errorf("command is not allowed: %v", job.Command)
	}

	execCtx := &CmdExecutorContext{
		Shell:      job.Shell,
		Command:    job.Command,
		WorkingDir: job.Cwd,
		IsSudo:     job.IsSudo,
		IsScript:   job.IsScript,
	}
	cmd := c.cmdExec.New(ctx, execCtx)
	stdOut := CapacityBuffer{capacity: c.config.RemoteCommands.SendBackLimit}
	stdErr := CapacityBuffer{capacity: c.config.RemoteCommands.SendBackLimit}
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	c.Debugf("Generated command is %s", cmd.String())

	startedAt := now()
	err = c.cmdExec.Start(cmd)
	if err != nil {
		c.runCmdMutex.Unlock()
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
		if job.IsScript {
			defer func() {
				err := os.Remove(job.Command)
				if err != nil {
					c.Errorf("failed to delete script %s: %v", job.Command, err)
				} else {
					c.Debugf("deleted script %s after execution", job.Command)
				}
			}()
		}

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
		now := now()
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

// var is used to override in tests
var getShell = func(inputShell, os string) (string, error) {
	if os == "windows" {
		switch inputShell {
		case "":
			return cmdShell, nil
		case cmdShell, powerShell:
			return inputShell, nil
		}
		return "", fmt.Errorf("invalid windows command shell: %q", inputShell)
	}

	if inputShell != "" {
		return "", fmt.Errorf("for unix clients a command shell should not be specified, got: %q", inputShell)
	}
	return unixShell, nil
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
