package chclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
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

func (e *CmdExecutorMock) New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
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
		_, err := cmd.Stdout.Write([]byte(s))
		if err != nil {
			log.Fatalf("Failed to write data into stdout: %s", err)
		}
	}
}

func (e *CmdExecutorMock) writeToStdErr(cmd *exec.Cmd) {
	defer e.wg.Done()

	for _, s := range e.ReturnStdErr {
		_, err := cmd.Stderr.Write([]byte(s))
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
	"client_id": "d81e6b93e75aef59a7701b90555f43808458b34e30370c3b808c1816a32252b3",
	"command": "/bin/date;foo;whoami",
	"created_by": "admin",
	"timeout_sec": 60,
	"sudo": true,
	"cwd": "/root"
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

func TestHandleRunCmdRequestPositiveCase(t *testing.T) {
	now = nowMockF
	assert := assert.New(t)

	// given
	getShell = func(inputShell, os string) (string, error) {
		return "test-shell", nil
	}
	wantPID := 123
	execMock := NewCmdExecutorMock()
	execMock.ReturnPID = wantPID
	execMock.ReturnStdOut = []string{"output1", "output2", "output3"}
	execMock.ReturnStdErr = []string{"error1", "error2"}
	connMock := test.NewConnMock()
	// mimic real behavior and wait until background task sends the request
	done := make(chan bool)
	connMock.DoneChannel = done
	configCopy := defaultValidMinConfig
	c := Client{
		cmdExec: execMock,
		sshConn: connMock,
		Logger:  testLog,
		config:  &configCopy,
	}

	wantJSONPart1 := `
{
	"jid": "5f02b216-3f8a-42be-b66c-f4c1d0ea3809",
	"status": "successful",
	"sudo": true,
	"is_script": false,
	"finished_at": "2020-08-19T12:00:00+03:00",
	"client_id": "d81e6b93e75aef59a7701b90555f43808458b34e30370c3b808c1816a32252b3",
	"client_name": "",
	"command": "/bin/date;foo;whoami",
	"shell": "test-shell",
	"pid": 123,
	"started_at": "2020-08-19T12:00:00+03:00",
	"created_by": "admin",
	"cwd": "/root",
	"timeout_sec": 60,
	"multi_job_id":null,
	"error":"",
`
	wantJSONPart2 := `
	   "result": {
			"stdout": "output1output2output3",
			"stderr": "error1error2"
		}
	}
	`
	stdOutSize := len(strings.Join(execMock.ReturnStdOut, ""))
	stdErrSize := len(strings.Join(execMock.ReturnStdErr, ""))

	testCases := []struct {
		name            string
		sendBackLimit   int
		denyRegexp      *regexp.Regexp
		wantJSON        string
		wantErrContains string
	}{
		{
			name:          "limit is larger than stdout and stderr",
			sendBackLimit: stdOutSize + 1,
			wantJSON:      wantJSONPart1 + wantJSONPart2,
		},
		{
			name:          "limit is equal to the larger output",
			sendBackLimit: stdOutSize,
			wantJSON:      wantJSONPart1 + wantJSONPart2,
		},
		{
			name:          "limit is equal to the smaller output",
			sendBackLimit: stdErrSize,
			wantJSON: wantJSONPart1 + `
        "result": {
        "stdout": "output1outpu",
        "stderr": "error1error2"
    }
}`,
		},
		{
			name:          "limit is less than smaller output",
			sendBackLimit: stdErrSize - 1,
			wantJSON: wantJSONPart1 + `
		"result": {
		"stdout": "output1outp",
		"stderr": "error1error"
	}
}`,
		},
		{
			name:          "limit is zero",
			sendBackLimit: 0,
			wantJSON: wantJSONPart1 + `
		"result": {
		"stdout": "",
		"stderr": ""
	}
}`,
		},
		{
			name:            "command is not allowed",
			denyRegexp:      regexp.MustCompile(".*"),
			wantErrContains: "command is not allowed",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			c.config.RemoteCommands.SendBackLimit = tc.sendBackLimit
			if tc.denyRegexp != nil {
				c.config.RemoteCommands.denyRegexp = []*regexp.Regexp{tc.denyRegexp}
			}

			// when
			res, err := c.HandleRunCmdRequest(context.Background(), []byte(jobToRunJSON))

			// then
			if tc.wantErrContains != "" {
				require.Error(t, err)
				assert.Contains(err.Error(), tc.wantErrContains)
				return
			}
			<-done

			// check returned result
			require.NoError(t, err)
			assert.Equal(&comm.RunCmdResponse{Pid: wantPID, StartedAt: nowMock}, res)

			// check job result that was sent to server
			inputRequestName, inputWantReply, inputPayload := connMock.InputSendRequest()
			assert.Equal(comm.RequestTypeCmdResult, inputRequestName)
			assert.Equal(false, inputWantReply)
			assert.JSONEq(tc.wantJSON, string(inputPayload))
		})
	}
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

	configCopy := defaultValidMinConfig
	c := Client{
		cmdExec: execMock,
		sshConn: connMock,
		Logger:  testLog,
		config:  &configCopy,
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

func TestRemoteCommandsDisabled(t *testing.T) {
	// given
	c := Client{
		Logger: testLog,
		config: &Config{
			RemoteCommands: CommandsConfig{
				Enabled: false,
			},
		},
	}

	// when
	gotRes, gotErr := c.HandleRunCmdRequest(context.Background(), []byte(jobToRunJSON))

	// then
	require.Error(t, gotErr)
	assert.Equal(t, "remote commands execution is disabled", gotErr.Error())
	assert.Nil(t, gotRes)
}

func TestIsCommandAllowed(t *testing.T) {
	defaultTestAllow := []string{"^/usr/bin.*", "^/usr/local/bin/.*", `^C:\\Windows\\System32.*`}
	testCases := []struct {
		name string

		cmd   string
		order [2]string
		allow []string
		deny  []string

		wantRes bool
	}{
		{
			name:    "allow-deny: does not match allow regexp",
			cmd:     "/some/cmd",
			order:   allowDenyOrder,
			allow:   defaultTestAllow,
			wantRes: false,
		},
		{
			name:    "allow-deny: matches both allow and deny regexp",
			cmd:     "/usr/bin/zip",
			order:   allowDenyOrder,
			allow:   defaultTestAllow,
			deny:    []string{"^/usr/bin/z.*"},
			wantRes: false,
		},
		{
			name:    "allow-deny: matches allow, empty deny",
			cmd:     "/usr/bin/zip",
			order:   allowDenyOrder,
			allow:   defaultTestAllow,
			deny:    []string{},
			wantRes: true,
		},
		{
			name:    "windows: allow-deny: matches allow, empty deny",
			cmd:     `C:\Windows\System32\some`,
			order:   allowDenyOrder,
			allow:   defaultTestAllow,
			deny:    []string{},
			wantRes: true,
		},
		{
			name:    "allow-deny: empty allow, does not match deny regexp",
			cmd:     "/bin/some/cmd",
			order:   allowDenyOrder,
			allow:   []string{},
			deny:    []string{"^/usr/bin/zip.*"},
			wantRes: false,
		},
		{
			name:    "allow-deny: matches allow regexp but not deny",
			cmd:     "/usr/bin/zip",
			order:   allowDenyOrder,
			allow:   defaultTestAllow,
			deny:    []string{"^/usr/bin/zip2.*", "zip3.*"},
			wantRes: true,
		},
		{
			name:    "allow-deny: does not match any regexp",
			cmd:     "/bin/some/cmd",
			order:   allowDenyOrder,
			allow:   defaultTestAllow,
			deny:    []string{"^/usr/bin/zip2.*"},
			wantRes: false,
		},
		{
			name:    "deny-allow: matches both deny and allow regexp",
			cmd:     "/usr/bin/zip",
			order:   denyAllowOrder,
			deny:    []string{"^/usr/bin/z.*"},
			allow:   defaultTestAllow,
			wantRes: true,
		},
		{
			name:    "deny-allow: matches deny regexp but not allow",
			cmd:     "/usr/test/test-cmd",
			order:   denyAllowOrder,
			deny:    []string{".*test.*"},
			allow:   []string{"^/usr/bin.*", "^/usr/local/bin/.*"},
			wantRes: false,
		},
		{
			name:    "deny-allow: matches allow regexp but not deny",
			cmd:     "/usr/bin/zip",
			order:   denyAllowOrder,
			deny:    []string{"^/usr/bin/zip2.*", ".*zip3.*"},
			allow:   defaultTestAllow,
			wantRes: true,
		},
		{
			name:    "allow-deny: does not match any regexp",
			cmd:     "/bin/some/cmd",
			order:   denyAllowOrder,
			allow:   defaultTestAllow,
			deny:    []string{"^/usr/bin/zip2.*"},
			wantRes: true,
		},
		{
			name:    "unknown order",
			cmd:     "/bin/some/cmd",
			order:   [2]string{"one", "two"},
			allow:   defaultTestAllow,
			deny:    []string{"^/usr/bin/zip.*"},
			wantRes: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			config := defaultValidMinConfig
			config.RemoteCommands.Deny = tc.deny
			c := Client{
				Logger: testLog,
				config: &config,
			}
			c.config.RemoteCommands.Order = tc.order
			c.config.RemoteCommands.allowRegexp = getRegexpList(tc.allow)
			c.config.RemoteCommands.denyRegexp = getRegexpList(tc.deny)

			// when
			gotRes := c.isAllowed(tc.cmd)

			// then
			assert.Equal(t, tc.wantRes, gotRes)
		})
	}
}

func getRegexpList(list []string) []*regexp.Regexp {
	var res []*regexp.Regexp
	for _, v := range list {
		res = append(res, regexp.MustCompile(v))
	}
	return res
}
