package chclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share"

	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type CmdExecutorContext struct {
	Interpreter string
	Command     string
	WorkingDir  string
	IsSudo      bool
	IsScript    bool
}

type CmdExecutor interface {
	New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd
	Start(cmd *exec.Cmd) error
	Wait(cmd *exec.Cmd) error
}

type CmdExecutorImpl struct {
	*chshare.Logger
}

func NewCmdExecutor(l *chshare.Logger) *CmdExecutorImpl {
	return &CmdExecutorImpl{
		Logger: l,
	}
}

func (e *CmdExecutorImpl) Start(cmd *exec.Cmd) error {
	return cmd.Start()
}

func (e *CmdExecutorImpl) Wait(cmd *exec.Cmd) error {
	return cmd.Wait()
}

const (
	unixShell  = "/bin/sh"
	cmdShell   = "cmd"
	powerShell = "powershell"
)

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

	if job.IsScript && !c.config.RemoteScripts.Enabled {
		return nil, errors.New("remote scripts are disabled")
	}

	// TODO: temporary solution, refactor with using worker pool
	c.runCmdMutex.Lock()

	job.Interpreter, err = getInterpreter(job.Interpreter, runtime.GOOS, job.HasShebang)
	if err != nil {
		c.runCmdMutex.Unlock()
		return nil, err
	}

	if !c.isAllowed(job.Command) {
		c.runCmdMutex.Unlock()
		return nil, fmt.Errorf("command is not allowed: %v", job.Command)
	}

	execCtx := &CmdExecutorContext{
		Interpreter: job.Interpreter,
		Command:     job.Command,
		WorkingDir:  job.Cwd,
		IsSudo:      job.IsSudo,
		IsScript:    job.IsScript,
	}
	cmd := c.cmdExec.New(ctx, execCtx)
	stdOut := &CapacityBuffer{capacity: c.config.RemoteCommands.SendBackLimit}
	stdErr := &CapacityBuffer{capacity: c.config.RemoteCommands.SendBackLimit}
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr

	c.Debugf("Generated command is %s, sysProcAttributes: %+v", cmd.String(), cmd.SysProcAttr)

	startedAt := now()
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

		// observing stopped - unset PID
		c.setCurCmdPID(nil)
		c.runCmdMutex.Unlock()

		// fill all unset fields
		now := now()
		job.FinishedAt = &now
		job.Status = status
		job.PID = &res.Pid
		job.StartedAt = startedAt

		job.Error = c.buildErrText(execErr, stdOut, stdErr)

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

// var is used to override in tests
var getInterpreter = func(inputInterpreter, os string, hasShebang bool) (string, error) {
	if os == "windows" {
		switch inputInterpreter {
		case "":
			return cmdShell, nil
		case cmdShell, powerShell:
			return inputInterpreter, nil
		}
		return "", fmt.Errorf("invalid windows command interpreter: %q", inputInterpreter)
	}

	if hasShebang {
		return "", nil
	}

	if inputInterpreter != "" {
		return "", fmt.Errorf("for unix clients a command interpreter should not be specified, got: %q", inputInterpreter)
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
		b.hasOverflow = true
		return 0, errors.New(b.GetOverflowMessage())
	}

	b.data = append(b.data, p...)

	return len(p), nil
}

func (b *CapacityBuffer) String() string {
	return string(b.data)
}
