//+build windows

package system

import (
	chshare "github.com/cloudradar-monitoring/rport/share"
)

func getInterpreterTestCases() []interpreterTestCase {
	return []interpreterTestCase{
		{
			name:         "empty",
			interpreter:  "",
			partialMatch: false,
			wantCmdStr:   chshare.CmdShell,
		},
		{
			name:            "cmd",
			interpreter:     chshare.CmdShell,
			partialMatch: false,
			wantCmdStr: "",
		},
		{
			name:            "powershell",
			interpreter:     chshare.PowerShell,
			partialMatch: false,
			wantCmdStr: "",
		},
		{
			name:            "invalid interpreter",
			interpreter:     "unsupported",
			partialMatch: false,
			wantCmdStr: "invalid windows interpreter",
		},
		{
			name:            "hasShebang, interpreter not empty",
			interpreter:     chshare.PowerShell,
			partialMatch: false,
			boolHasShebang:  true,
			wantCmdStr: "",
		},
		{
			name:            "tacoscript interpreter",
			interpreter:     chshare.Tacoscript,
			wantCmdStr: chshare.Tacoscript,
		},
		{
			name:               "interpreter aliases",
			interpreter:        "pwsh7",
			wantCmdStr:    `C:\Program Files\PowerShell\7\pwsh.exe`,
			interpreterAliases: map[string]string{"pwsh7": `C:\Program Files\PowerShell\7\pwsh.exe`},
		},
		{
			name:            "interpreter full path",
			interpreter:     `C:\Program Files\Git\bin\bash.exe`,
			wantCmdStr: `C:\Program Files\Git\bin\bash.exe`,
		},
		{
			name:            "interpreter with params",
			interpreter:     `C:\Program Files\PowerShell\7\pwsh.exe -Noninteractive -executionpolicy bypass -File`,
			wantCmdStr: `C:\Program Files\PowerShell\7\pwsh.exe -Noninteractive -executionpolicy bypass -File`,
		},
	}
}
