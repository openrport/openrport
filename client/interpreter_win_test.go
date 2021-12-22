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
			wantInterpreter: chshare.CmdShell,
			wantErrContains: "",
		},
		{
			name:            "cmd",
			interpreter:     chshare.CmdShell,
			wantInterpreter: chshare.CmdShell,
			wantErrContains: "",
		},
		{
			name:            "powershell",
			interpreter:     chshare.PowerShell,
			wantInterpreter: chshare.PowerShell,
			wantErrContains: "",
		},
		{
			name:            "invalid interpreter",
			interpreter:     "unsupported",
			wantInterpreter: "",
			wantErrContains: "invalid windows interpreter",
		},
		{
			name:            "hasShebang, interpreter not empty",
			interpreter:     chshare.PowerShell,
			wantInterpreter: chshare.PowerShell,
			boolHasShebang:  true,
		},
		{
			name:            "tacoscript interpreter",
			interpreter:     chshare.Tacoscript,
			wantInterpreter: chshare.Tacoscript,
		},
		{
			name:               "interpreter aliases",
			interpreter:        "pwsh7",
			wantInterpreter:    `C:\Program Files\PowerShell\7\pwsh.exe`,
			interpreterAliases: map[string]string{"pwsh7": `C:\Program Files\PowerShell\7\pwsh.exe`},
		},
		{
			name:            "interpreter full path",
			interpreter:     `C:\Program Files\Git\bin\bash.exe`,
			wantInterpreter: `C:\Program Files\Git\bin\bash.exe`,
		},
	}
}
