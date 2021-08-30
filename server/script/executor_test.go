package script

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"
)

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

	cl := &clients.Client{
		OSKernel:   "linux",
		ID:         "123",
		Connection: conn,
	}
	inp := &api.ExecuteInput{
		Script:     "pwd",
		Cwd:        "/home",
		TimeoutSec: 1,
	}

	res, err := executor.CreateScriptOnClient(inp, cl)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/script.sh", res)

	fileInput := &models.File{}
	_, _, fileInputBytes := conn.InputSendRequest()

	err = json.Unmarshal(fileInputBytes, fileInput)
	require.NoError(t, err)

	assert.True(t, strings.HasSuffix(fileInput.Name, ".sh"))
	assert.Equal(t, "pwd", string(fileInput.Content))
	assert.EqualValues(t, 0744, fileInput.Mode)
}

func TestParsingShebangLine(t *testing.T) {
	withShebang := `#!/bin/cat
Hello world!`
	withoutShebang := "echo 123"

	assert.True(t, HasShebangLine(withShebang))
	assert.False(t, HasShebangLine(withoutShebang))
}
