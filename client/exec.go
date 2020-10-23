package chclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type CmdExecutor interface {
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
		"-Command",
	},
}

// now is used to stub time.Now in tests
var now = time.Now

func (c *Client) HandleRunCmdRequest(ctx context.Context, reqPayload []byte) (*comm.RunCmdResponse, error) {
	// do not accept a new request when the previous is not finished yet
	// NOTE: HandleRunCmdRequest is run sequentially, that's why no need to lock a block with read/write curPID
	curPID := c.getCurCmdPID()
	if curPID != nil {
		return nil, fmt.Errorf("a previous command execution with PID %d is still running", *curPID)
	}

	job := models.Job{}
	err := json.Unmarshal(reqPayload, &job)
	if err != nil {
		return nil, fmt.Errorf("failed to decode requested job: %s", err)
	}
	job.Shell, err = getShell(job.Shell, runtime.GOOS)
	if err != nil {
		return nil, err
	}

	var args []string
	args = append(args, shellOptions[job.Shell]...)
	args = append(args, job.Command)

	cmd := exec.CommandContext(ctx, job.Shell, args...)
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	startedAt := now()
	err = c.cmdExec.Start(cmd)
	if err != nil {
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

		// fill all unset fields
		now := now()
		job.FinishedAt = &now
		job.Status = status
		job.PID = res.Pid
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
