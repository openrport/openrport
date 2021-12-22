package chclient

import (
	"testing"

	chshare "github.com/cloudradar-monitoring/rport/share"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInterpreter(t *testing.T) {
	win := "windows"
	unix := "linux"
	testCases := []struct {
		name               string
		interpreter        string
		os                 string
		wantInterpreter    string
		wantErrContains    string
		boolHasShebang     bool
		interpreterAliases map[string]string
	}{
		{
			name:            "windows, empty",
			interpreter:     "",
			os:              win,
			wantInterpreter: chshare.CmdShell,
			wantErrContains: "",
		},
		{
			name:            "windows, cmd",
			interpreter:     chshare.CmdShell,
			os:              win,
			wantInterpreter: chshare.CmdShell,
			wantErrContains: "",
		},
		{
			name:            "windows, powershell",
			interpreter:     chshare.PowerShell,
			os:              win,
			wantInterpreter: chshare.PowerShell,
			wantErrContains: "",
		},
		{
			name:            "windows, invalid interpreter",
			interpreter:     "unsupported",
			os:              win,
			wantInterpreter: "",
			wantErrContains: "invalid windows interpreter",
		},
		{
			name:            "unix, empty",
			interpreter:     "",
			os:              unix,
			wantInterpreter: chshare.UnixShell,
			wantErrContains: "",
		},
		{
			name:            "unix, non empty",
			interpreter:     chshare.UnixShell,
			os:              unix,
			wantInterpreter: chshare.UnixShell,
		},
		{
			name:            "empty os, empty interpreter",
			interpreter:     "",
			os:              "",
			wantInterpreter: chshare.UnixShell,
			wantErrContains: "",
		},
		{
			name:            "unix, hasShebang, interpreter empty",
			os:              unix,
			wantInterpreter: "",
			boolHasShebang:  true,
		},
		{
			name:            "unix, hasShebang, interpreter not empty",
			os:              unix,
			interpreter:     chshare.UnixShell,
			wantInterpreter: "",
			boolHasShebang:  true,
		},
		{
			name:            "windows, hasShebang, interpreter not empty",
			os:              win,
			interpreter:     chshare.PowerShell,
			wantInterpreter: chshare.PowerShell,
			boolHasShebang:  true,
		},
		{
			name:            "windows, tacoscript interpreter",
			os:              win,
			interpreter:     chshare.Tacoscript,
			wantInterpreter: chshare.Tacoscript,
		},
		{
			name:            "linux, tacoscript interpreter",
			os:              unix,
			interpreter:     chshare.Tacoscript,
			wantInterpreter: chshare.Tacoscript,
		},
		{
			name:               "linux, interpreter aliases",
			os:                 unix,
			interpreter:        "taco",
			wantInterpreter:    chshare.Tacoscript,
			interpreterAliases: map[string]string{"taco": chshare.Tacoscript},
		},
		{
			name:               "win, interpreter aliases",
			os:                 win,
			interpreter:        "pwsh7",
			wantInterpreter:    `C:\Program Files\PowerShell\7\pwsh.exe`,
			interpreterAliases: map[string]string{"pwsh7": `C:\Program Files\PowerShell\7\pwsh.exe`},
		},
		//{
		//	name:            "win, interpreter full path",
		//	os:              win,
		//	interpreter:     `C:\Program Files\Git\bin\bash.exe`,
		//	wantInterpreter: `C:\Program Files\Git\bin\bash.exe`,
		//},
		{
			name:            "linux, interpreter full path",
			os:              unix,
			interpreter:     `/usr/local/bin/bash`,
			wantInterpreter: `/usr/local/bin/bash`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			interpreterInput := interpreterProviderInput{
				name:       tc.interpreter,
				hasShebang: tc.boolHasShebang,
				aliasesMap: tc.interpreterAliases,
				os:         tc.os,
			}
			gotInterpreter, gotErr := getInterpreter(interpreterInput)

			// then
			if len(tc.wantErrContains) > 0 {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, gotErr)
				assert.Equal(t, tc.wantInterpreter, gotInterpreter)
			}
		})
	}
}
