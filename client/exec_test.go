package chclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/test"
)

type CmdExecutorMock struct {
	DoneChannel    chan bool
	ReturnPID      int
	ReturnStartErr error
	ReturnWaitErr  error
	ReturnStdOut   []string
	ReturnStdErr   []string

	wg sync.WaitGroup
}

func NewCmdExecutorMock() *CmdExecutorMock {
	return &CmdExecutorMock{}
}

func (e *CmdExecutorMock) New(ctx context.Context, shell, command string) *exec.Cmd {
	var args []string
	args = append(args, shellOptions[shell]...)
	args = append(args, command)
	cmd := exec.CommandContext(ctx, shell, args...)
	return cmd
}

func (e *CmdExecutorMock) Start(cmd *exec.Cmd) error {
	if e.ReturnStartErr != nil {
		return e.ReturnStartErr
	}

	if e.ReturnPID != 0 {
		cmd.Process = &os.Process{Pid: e.ReturnPID}
	}

	// mock output to stdout and stderr
	e.wg.Add(1)
	go e.writeToStdOut(cmd)
	e.wg.Add(1)
	go e.writeToStdErr(cmd)

	return nil
}

func (e *CmdExecutorMock) writeToStdOut(cmd *exec.Cmd) {
	defer e.wg.Done()

	for _, s := range e.ReturnStdOut {
		_, err := cmd.Stdout.Write([]byte(s + "\n"))
		if err != nil {
			log.Fatalf("Failed to write data into stdout: %s", err)
		}
	}
}

func (e *CmdExecutorMock) writeToStdErr(cmd *exec.Cmd) {
	defer e.wg.Done()

	for _, s := range e.ReturnStdErr {
		_, err := cmd.Stderr.Write([]byte(s + "\n"))
		if err != nil {
			log.Fatalf("Failed to write data into stderr: %s", err)
		}
	}
}

func (e *CmdExecutorMock) Wait(cmd *exec.Cmd) error {
	if e.ReturnWaitErr != nil {
		return e.ReturnWaitErr
	}
	e.wg.Wait()
	// wait if needed
	if e.DoneChannel != nil {
		e.DoneChannel <- true
	}
	return nil
}

// nowMock is used to override time now.
var nowMockF = func() time.Time {
	n, _ := time.Parse(time.RFC3339, "2020-08-19T12:00:00+03:00")
	return n
}

var nowMock = nowMockF()

var testLog = chshare.NewLogger("client", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)

const jobToRunJSON = `
{
	"jid": "5f02b216-3f8a-42be-b66c-f4c1d0ea3809",
	"sid": "d81e6b93e75aef59a7701b90555f43808458b34e30370c3b808c1816a32252b3",
	"command": "/bin/date;foo;whoami",
	"created_by": "admin",
	"timeout_sec": 60
}
`

func TestGetShell(t *testing.T) {
	win := "windows"
	unix := "linux"
	testCases := []struct {
		name            string
		shell           string
		os              string
		wantShell       string
		wantErrContains string
	}{
		{
			name:            "windows, empty",
			shell:           "",
			os:              win,
			wantShell:       cmdShell,
			wantErrContains: "",
		},
		{
			name:            "windows, cmd",
			shell:           cmdShell,
			os:              win,
			wantShell:       cmdShell,
			wantErrContains: "",
		},
		{
			name:            "windows, powershell",
			shell:           powerShell,
			os:              win,
			wantShell:       powerShell,
			wantErrContains: "",
		},
		{
			name:            "windows, invalid shell",
			shell:           "unsupported",
			os:              win,
			wantShell:       "",
			wantErrContains: "invalid windows command shell",
		},
		{
			name:            "unix, empty",
			shell:           "",
			os:              unix,
			wantShell:       unixShell,
			wantErrContains: "",
		},
		{
			name:            "unix, non empty",
			shell:           unixShell,
			os:              unix,
			wantShell:       "",
			wantErrContains: "for unix clients a command shell should not be specified",
		},
		{
			name:            "empty os, empty shell",
			shell:           "",
			os:              "",
			wantShell:       unixShell,
			wantErrContains: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotShell, gotErr := getShell(tc.shell, tc.os)

			// then
			if len(tc.wantErrContains) > 0 {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, gotErr)
				assert.Equal(t, tc.wantShell, gotShell)
			}
		})
	}
}

func TestHandleRunCmdRequestNormalCase(t *testing.T) {
	now = nowMockF
	assert := assert.New(t)

	// given
	getShell = func(inputShell, os string) (string, error) {
		return "test-shell", nil
	}
	wantPID := 123
	execMock := NewCmdExecutorMock()
	execMock.ReturnPID = wantPID
	execMock.ReturnStdOut = []string{"1", "2", "3"}
	execMock.ReturnStdErr = []string{"error1", "error2"}
	connMock := test.NewConnMock()
	// mimic real behavior and wait until background task sends the request
	done := make(chan bool)
	connMock.DoneChannel = done
	c := Client{
		cmdExec: execMock,
		sshConn: connMock,
		Logger:  testLog,
	}

	wantJobJSON := `
{
	"jid": "5f02b216-3f8a-42be-b66c-f4c1d0ea3809",
	"status": "successful",
	"finished_at": "2020-08-19T12:00:00+03:00",
	"sid": "d81e6b93e75aef59a7701b90555f43808458b34e30370c3b808c1816a32252b3",
	"command": "/bin/date;foo;whoami",
	"shell": "test-shell",
	"pid": 123,
	"started_at": "2020-08-19T12:00:00+03:00",
	"created_by": "admin",
	"timeout_sec": 60,
	"result": {
		"stdout": "1\n2\n3\n",
		"stderr": "error1\nerror2\n"
	}
}

`
	// when
	res, err := c.HandleRunCmdRequest(context.Background(), []byte(jobToRunJSON))
	<-done

	// then

	// check returned result
	require.NoError(t, err)
	assert.Equal(&comm.RunCmdResponse{Pid: wantPID, StartedAt: nowMock}, res)

	// check job result that was sent to server
	inputRequestName, inputWantReply, inputPayload := connMock.InputSendRequest()
	assert.Equal(comm.RequestTypeCmdResult, inputRequestName)
	assert.Equal(false, inputWantReply)
	assert.JSONEq(wantJobJSON, string(inputPayload))
}

func TestHandleRunCmdRequestHasRunningCmd(t *testing.T) {
	now = nowMockF
	assert := assert.New(t)

	// given
	wantPID := 123
	execMock := NewCmdExecutorMock()
	execMock.ReturnPID = wantPID
	execMock.ReturnStdOut = []string{"1", "2", "3"}
	execMock.ReturnStdErr = []string{"error1", "error2"}

	connMock := test.NewConnMock()

	// mimic real behavior to have the 1st command still running when the 2nd request comes
	doneSendResp := make(chan bool)
	connMock.DoneChannel = doneSendResp
	doneCmd := make(chan bool)
	execMock.DoneChannel = doneCmd

	c := Client{
		cmdExec: execMock,
		sshConn: connMock,
		Logger:  testLog,
	}

	// when
	// run two cmds to get an error for the 2nd
	res1, err1 := c.HandleRunCmdRequest(context.Background(), []byte(jobToRunJSON))
	res2, err2 := c.HandleRunCmdRequest(context.Background(), []byte(jobToRunJSON))

	// then
	// check that running new commands is blocked
	curPID := c.getCurCmdPID()
	require.NotNil(t, curPID)
	assert.Equal(wantPID, *curPID)
	// finish the cmd execution
	<-doneCmd
	// finish to send the response to server
	<-doneSendResp
	// check that running new commands is not blocked anymore
	curPID = c.getCurCmdPID()
	assert.Nil(curPID)

	// check the result
	require.NoError(t, err1)
	assert.Equal(&comm.RunCmdResponse{Pid: wantPID, StartedAt: nowMock}, res1)
	assert.Error(err2)
	assert.Equal(fmt.Errorf("a previous command execution with PID %d is still running", wantPID), err2)
	assert.Nil(res2)
}
