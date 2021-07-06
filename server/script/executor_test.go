package script

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"
)

func TestConvertScriptInputToCmdInput(t *testing.T) {
	testCases := []struct {
		name                   string
		oskernel               string
		clientID               string
		isPowershell           bool
		isSudo                 bool
		cwd                    string
		scriptPath             string
		timeout                time.Duration
		epxectedCommand        string
		expectedSecondsTimeout int
		expectedShell          string
	}{
		{
			name:                   "windows powershell on",
			oskernel:               "windows",
			clientID:               "213",
			isPowershell:           true,
			cwd:                    "C:\\",
			scriptPath:             "C:\\script.sh",
			timeout:                time.Second,
			epxectedCommand:        "C:\\script.sh",
			expectedSecondsTimeout: 1,
			expectedShell:          "powershell",
		},
		{
			name:                   "windows powershell off",
			oskernel:               "windows",
			clientID:               "214",
			isPowershell:           false,
			cwd:                    "C:\\",
			scriptPath:             "C:\\script.sh",
			timeout:                time.Second * 2,
			epxectedCommand:        "C:\\script.sh",
			expectedSecondsTimeout: 2,
			expectedShell:          "cmd",
		},
		{
			name:                   "linux sudo",
			oskernel:               "linux",
			clientID:               "215",
			isSudo:                 true,
			cwd:                    "/root/here",
			scriptPath:             "/tmp/script.sh",
			timeout:                time.Minute,
			epxectedCommand:        "/tmp/script.sh",
			expectedSecondsTimeout: 60,
		},
	}

	for i := range testCases {
		t.Run(testCases[i].name, func(t *testing.T) {
			ipt := &ExecutionInput{
				Client: &clients.Client{
					OSKernel: testCases[i].oskernel,
					ID:       testCases[i].clientID,
				},
				IsPowershell: testCases[i].isPowershell,
				IsSudo:       testCases[i].isSudo,
				Cwd:          testCases[i].cwd,
				Timeout:      testCases[i].timeout,
			}

			executor := NewExecutor(&chshare.Logger{})
			res := executor.ConvertScriptInputToCmdInput(ipt, testCases[i].scriptPath)

			assert.Equal(t, testCases[i].epxectedCommand, res.Command)
			assert.Equal(t, testCases[i].expectedSecondsTimeout, res.TimeoutSec)
			assert.Equal(t, testCases[i].cwd, res.Cwd)
			assert.Equal(t, testCases[i].expectedShell, res.Shell)
			assert.Equal(t, testCases[i].isSudo, res.IsSudo)
			assert.Equal(t, testCases[i].clientID, res.ClientID)
		})
	}
}

func TestCreateScriptOnClient(t *testing.T) {
	givenResp := &comm.CreateFileResponse{
		FilePath:   "/tmp/script.sh",
		Sha256Hash: "a1159e9df3670d549d04524532629f5477ceb7deec9b45e47e8c009506ecb2c8", //sha256 of "pwd"
		CreatedAt:  time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	givenRespBytes, err := json.Marshal(givenResp)
	require.NoError(t, err)

	executor := NewExecutor(&chshare.Logger{})
	conn := &test.ConnMock{
		ReturnResponsePayload: givenRespBytes,
		ReturnOk:              true,
	}
	inp := &ExecutionInput{
		Client: &clients.Client{
			OSKernel:   "linux",
			ID:         "123",
			Connection: conn,
		},
		ScriptBody: []byte("pwd"),
		Cwd:        "/home",
		Timeout:    time.Second,
	}

	res, err := executor.CreateScriptOnClient(inp)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/script.sh", res)

	fileInput := &models.File{}
	_, _, fileInputBytes := conn.InputSendRequest()

	err = json.Unmarshal(fileInputBytes, fileInput)
	require.NoError(t, err)

	assert.True(t, strings.HasSuffix(fileInput.Name, ".sh"))
	assert.Equal(t, "pwd", string(fileInput.Content))
	assert.EqualValues(t, 0744, fileInput.Mode)
	assert.True(t, fileInput.CreateDir)
}
