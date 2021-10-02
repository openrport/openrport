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
		name                 string
		filePathWant         string
		Sha256HashToMocked   string
		clientOSKernelMocked string
		scriptMocked         string
		fileExtensionWant    string
		interpreterMocked    string
	}{
		{
			name:                 "sh script linux",
			filePathWant:         "/tmp/script.sh",
			Sha256HashToMocked:   defaultHash,
			clientOSKernelMocked: "linux",
			scriptMocked:         defaultScript,
			fileExtensionWant:    ".sh",
		},
		{
			name:                 "tacoscript script linux",
			interpreterMocked:    chshare.Tacoscript,
			filePathWant:         "/tmp/taco_script.yml",
			Sha256HashToMocked:   defaultHash,
			clientOSKernelMocked: "linux",
			scriptMocked:         defaultScript,
			fileExtensionWant:    ".yml",
		},
		{
			name:                 "tacoscript windows",
			interpreterMocked:    chshare.Tacoscript,
			filePathWant:         "C:\\taco_script.yml",
			Sha256HashToMocked:   defaultHash,
			clientOSKernelMocked: "windows",
			scriptMocked:         defaultScript,
			fileExtensionWant:    ".yml",
		},
		{
			name:                 "powershell script windows",
			interpreterMocked:    chshare.PowerShell,
			filePathWant:         "C:\\ps_script.ps1",
			Sha256HashToMocked:   defaultHash,
			clientOSKernelMocked: "windows",
			scriptMocked:         defaultScript,
			fileExtensionWant:    ".ps1",
		},
		{
			name:                 "cmd script windows",
			interpreterMocked:    chshare.CmdShell,
			filePathWant:         "C:\\cmd_script.bat",
			Sha256HashToMocked:   defaultHash,
			clientOSKernelMocked: "windows",
			scriptMocked:         defaultScript,
			fileExtensionWant:    ".bat",
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			givenResp := &comm.CreateFileResponse{
				FilePath:   tc.filePathWant,
				Sha256Hash: tc.Sha256HashToMocked,
			}
			givenRespBytes, err := json.Marshal(givenResp)
			require.NoError(t, err)

			executor := NewExecutor(&chshare.Logger{})
			conn := &test.ConnMock{
				ReturnResponsePayload: givenRespBytes,
				ReturnOk:              true,
			}

			cl := &clients.Client{
				OSKernel:   tc.clientOSKernelMocked,
				Connection: conn,
			}
			inp := &api.ExecuteInput{
				Script:      tc.scriptMocked,
				Interpreter: tc.interpreterMocked,
			}

			gotScriptPath, err := executor.CreateScriptOnClient(inp, cl)
			require.NoError(t, err)
			assert.Equal(t, tc.filePathWant, gotScriptPath)

			fileInputGot := &models.File{}
			_, _, fileInputBytes := conn.InputSendRequest()

			err = json.Unmarshal(fileInputBytes, fileInputGot)
			require.NoError(t, err)

			assert.True(t, strings.HasSuffix(fileInputGot.Name, tc.fileExtensionWant))
			assert.Equal(t, tc.scriptMocked, string(fileInputGot.Content))
			assert.EqualValues(t, DefaultScriptFileMode, fileInputGot.Mode)
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
