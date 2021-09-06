package script

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateScriptOnClient(t *testing.T) {
	const defaultHash = "a1159e9df3670d549d04524532629f5477ceb7deec9b45e47e8c009506ecb2c8" //sha256 of "pwd"
	const defaultScript = "pwd"
	testCases := []struct {
		name                  string
		filePathToReturn      string
		Sha256HashToReturn    string
		clientOSKernelToGive  string
		scriptToGive          string
		fileExtensionToExpect string
		interpreterToGive     string
	}{
		{
			name:                  "sh script linux",
			filePathToReturn:      "/tmp/script.sh",
			Sha256HashToReturn:    defaultHash,
			clientOSKernelToGive:  "linux",
			scriptToGive:          defaultScript,
			fileExtensionToExpect: ".sh",
		},
		{
			name:                  "taco script linux",
			interpreterToGive:     chshare.Taco,
			filePathToReturn:      "/tmp/taco_script.yml",
			Sha256HashToReturn:    defaultHash,
			clientOSKernelToGive:  "linux",
			scriptToGive:          defaultScript,
			fileExtensionToExpect: ".yml",
		},
		{
			name:                  "taco script windows",
			interpreterToGive:     chshare.Taco,
			filePathToReturn:      "C:\\taco_script.yml",
			Sha256HashToReturn:    defaultHash,
			clientOSKernelToGive:  "windows",
			scriptToGive:          defaultScript,
			fileExtensionToExpect: ".yml",
		},
		{
			name:                  "powershell script windows",
			interpreterToGive:     chshare.PowerShell,
			filePathToReturn:      "C:\\ps_script.ps1",
			Sha256HashToReturn:    defaultHash,
			clientOSKernelToGive:  "windows",
			scriptToGive:          defaultScript,
			fileExtensionToExpect: ".ps1",
		},
		{
			name:                  "cmd script windows",
			interpreterToGive:     chshare.CmdShell,
			filePathToReturn:      "C:\\cmd_script.bat",
			Sha256HashToReturn:    defaultHash,
			clientOSKernelToGive:  "windows",
			scriptToGive:          defaultScript,
			fileExtensionToExpect: ".bat",
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			givenResp := &comm.CreateFileResponse{
				FilePath:   tc.filePathToReturn,
				Sha256Hash: tc.Sha256HashToReturn,
			}
			givenRespBytes, err := json.Marshal(givenResp)
			require.NoError(t, err)

			executor := NewExecutor(&chshare.Logger{})
			conn := &test.ConnMock{
				ReturnResponsePayload: givenRespBytes,
				ReturnOk:              true,
			}

			cl := &clients.Client{
				OSKernel:   tc.clientOSKernelToGive,
				Connection: conn,
			}
			inp := &api.ExecuteInput{
				Script:      tc.scriptToGive,
				Interpreter: tc.interpreterToGive,
			}

			res, err := executor.CreateScriptOnClient(inp, cl)
			require.NoError(t, err)
			assert.Equal(t, tc.filePathToReturn, res)

			fileInputGiven := &models.File{}
			_, _, fileInputBytes := conn.InputSendRequest()

			err = json.Unmarshal(fileInputBytes, fileInputGiven)
			require.NoError(t, err)

			assert.True(t, strings.HasSuffix(fileInputGiven.Name, tc.fileExtensionToExpect))
			assert.Equal(t, tc.scriptToGive, string(fileInputGiven.Content))
			assert.EqualValues(t, DefaultScriptFileMode, fileInputGiven.Mode)
		})
	}
}

func TestParsingShebangLine(t *testing.T) {
	withShebang := `#!/bin/cat
Hello world!`
	withoutShebang := "echo 123"

	assert.True(t, HasShebangLine(withShebang))
	assert.False(t, HasShebangLine(withoutShebang))
}
