package script

import (
	"encoding/json"
	"testing"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
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
		clientOSKernelMocked string
		interpreterMocked    string
	}{
		{
			name:                 "sh script linux",
			filePathWant:         "/tmp/script.sh",
			clientOSKernelMocked: "linux",
		},
		{
			name:                 "taco script linux",
			interpreterMocked:    chshare.Taco,
			filePathWant:         "/tmp/taco_script.yml",
			clientOSKernelMocked: "linux",
		},
		{
			name:                 "taco script windows",
			interpreterMocked:    chshare.Taco,
			filePathWant:         "C:\\taco_script.yml",
			clientOSKernelMocked: "windows",
		},
		{
			name:                 "powershell script windows",
			interpreterMocked:    chshare.PowerShell,
			filePathWant:         "C:\\ps_script.ps1",
			clientOSKernelMocked: "windows",
		},
		{
			name:                 "cmd script windows",
			interpreterMocked:    chshare.CmdShell,
			filePathWant:         "C:\\cmd_script.bat",
			clientOSKernelMocked: "windows",
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			givenResp := &comm.CreateFileResponse{
				FilePath:   tc.filePathWant,
				Sha256Hash: defaultHash,
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
				Script:      defaultScript,
				Interpreter: tc.interpreterMocked,
			}

			gotScriptPath, err := executor.CreateScriptOnClient(inp, cl)
			require.NoError(t, err)
			assert.Equal(t, tc.filePathWant, gotScriptPath)
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
