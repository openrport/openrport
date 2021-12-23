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
			partialMatch: true,
			wantCmdStr:   "cmd.exe",
		},
		{
			name:         "cmd",
			interpreter:  chshare.CmdShell,
			partialMatch: true,
			wantCmdStr:   "cmd.exe",
		},
		{
			name:         "powershell",
			interpreter:  chshare.PowerShell,
			partialMatch: true,
			command:      `C:\\script.ps1`,
			wantCmdStr:   `powershell.exe -Noninteractive -executionpolicy bypass -File C:\\script.ps1`,
		},
		{
			name:           "hasShebang, interpreter not empty",
			interpreter:    chshare.PowerShell,
			partialMatch:   true,
			boolHasShebang: true,
			command:        `C:\\script.ps1`,
			wantCmdStr:     `powershell.exe -Noninteractive -executionpolicy bypass -File C:\\script.ps1`,
		},
		{
			name:        "tacoscript interpreter",
			interpreter: chshare.Tacoscript,
			command:     `C:\\script.ps1`,
			wantCmdStr:  `tacoscript.exe C:\\script.ps1`,
		},
		{
			name:               "interpreter aliases",
			interpreter:        "pwsh7",
			command:            `C:\\script.ps1`,
			interpreterAliases: map[string]string{"pwsh7": `C:\Program Files\PowerShell\7\pwsh.exe`},
			wantCmdStr:         `C:\Program Files\PowerShell\7\pwsh.exe C:\\script.ps1`,
		},
		{
			name:        "interpreter full path",
			command:     `C:\\script.ps1`,
			interpreter: `C:\Program Files\Git\bin\bash.exe`,
			wantCmdStr:  `C:\Program Files\Git\bin\bash.exe C:\\script.ps1`,
		},
		{
			name:        "interpreter with params",
			interpreter: `C:\Program Files\PowerShell\7\pwsh.exe -Noninteractive -executionpolicy bypass -File`,
			command:     `C:\\script.ps1`,
			wantCmdStr:  `C:\Program Files\PowerShell\7\pwsh.exe -Noninteractive -executionpolicy bypass -File C:\\script.ps1`,
		},
	}
}
