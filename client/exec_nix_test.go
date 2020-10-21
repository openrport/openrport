//+build !windows

package chclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"
)

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
	"timeout": 60000000000
}
`

func TestHandleRunCmdRequestNormalCase(t *testing.T) {
	now = nowMockF
	assert := assert.New(t)

	// given
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
	"pid": 123,
	"started_at": "2020-08-19T12:00:00+03:00",
	"created_by": "admin",
	"timeout": 60000000000,
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
	wantJob := parseJob(t, wantJobJSON)
	gotJob := parseJob(t, string(inputPayload))
	assert.Equal(wantJob, gotJob)
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

func parseJob(t *testing.T, jobJSON string) models.Job {
	job := models.Job{}
	err := json.Unmarshal([]byte(jobJSON), &job)
	assert.NoError(t, err)
	return job
}
