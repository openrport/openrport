//+build !windows

package chclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
)

// now is used to stub time.Now in tests
var now = func() time.Time {
	return time.Now()
}

func (c *Client) HandleRunCmdRequest(ctx context.Context, reqPayload []byte) (*comm.RunCmdResponse, error) {
	// do not accept a new request when the previous is not finished yet
	// NOTE: HandleRunCmdRequest is run sequentially, that's why no need to lock a block with read/write curPID
	curPID := c.getCurCmdPID()
	if curPID != nil {
		return nil, fmt.Errorf("a previous command execution with PID %d is still running", *curPID)
	}

	job := models.Job{}
	if err := json.Unmarshal(reqPayload, &job); err != nil {
		return nil, fmt.Errorf("failed to decode requested job: %s", err)
	}

	// in order to run multiple commands under one process run with "/bin/sh -c <cmd1;cmd2;...>"
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", job.Command)
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	startedAt := now()
	err := c.cmdExec.Start(cmd)
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
				status = models.JobStatusFinished
			}
		case <-time.After(job.Timeout):
			status = models.JobStatusUnknown
			c.Debugf("timeout %s reached, stop observing command[jid=%q,pid=%d]:\n%s", job.Timeout, job.JID, res.Pid, job.Command)
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
		}
		_, _, err = c.sshConn.SendRequest(comm.RequestTypeCmdResult, false, jobBytes)
		if err != nil {
			c.Errorf("failed to send command result to server[jid=%q,pid=%d]: %s", job.JID, res.Pid, err)
		}

		c.Debugf("finished to observe cmd [jid=%q,pid=%d]", job.JID, res.Pid)
	}()

	return res, nil
}
