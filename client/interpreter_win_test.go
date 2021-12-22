//+build windows

package chclient

import (
	chshare "github.com/cloudradar-monitoring/rport/share"
)

func getInterpreterTestCases() []interpreterTestCase {
	return []interpreterTestCase{
		{
			name:            "empty",
			interpreter:     "",
			os:              win,
			wantInterpreter: chshare.CmdShell,
			wantErrContains: "",
		},
		{
			name:            "cmd",
			interpreter:     chshare.CmdShell,
			os:              win,
			wantInterpreter: chshare.CmdShell,
			wantErrContains: "",
		},
		{
			name:            "powershell",
			interpreter:     chshare.PowerShell,
			os:              win,
			wantInterpreter: chshare.PowerShell,
			wantErrContains: "",
		},
		{
			name:            "invalid interpreter",
			interpreter:     "unsupported",
			os:              win,
			wantInterpreter: "",
			wantErrContains: "invalid windows interpreter",
		},
		{
			name:            "hasShebang, interpreter not empty",
			os:              win,
			interpreter:     chshare.PowerShell,
			wantInterpreter: chshare.PowerShell,
			boolHasShebang:  true,
		},
		{
			name:            "tacoscript interpreter",
			os:              win,
			interpreter:     chshare.Tacoscript,
			wantInterpreter: chshare.Tacoscript,
		},
		{
			name:               "interpreter aliases",
			os:                 win,
			interpreter:        "pwsh7",
			wantInterpreter:    `C:\Program Files\PowerShell\7\pwsh.exe`,
			interpreterAliases: map[string]string{"pwsh7": `C:\Program Files\PowerShell\7\pwsh.exe`},
		},
		{
			name:            "interpreter full path",
			os:              win,
			interpreter:     `C:\Program Files\Git\bin\bash.exe`,
			wantInterpreter: `C:\Program Files\Git\bin\bash.exe`,
		},
	}
}
