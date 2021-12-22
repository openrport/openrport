//+build !windows

package system

import (
	chshare "github.com/cloudradar-monitoring/rport/share"
)

func getInterpreterTestCases() []interpreterTestCase {
	return []interpreterTestCase{
		{
			name:        "empty",
			interpreter: chshare.UnixShell,
			wantCmdStr:  "/bin/sh -c /script.sh",
		},
		{
			name:        "non empty sh",
			interpreter: chshare.UnixShell,
			wantCmdStr:  "/bin/sh -c /script.sh",
		},
		{
			name:           "hasShebang, interpreter empty",
			interpreter:    "",
			boolHasShebang: true,
			wantCmdStr:     "/script.sh",
		},
		{
			name:           "hasShebang, interpreter not empty",
			interpreter:    chshare.UnixShell,
			wantCmdStr:     "/script.sh",
			boolHasShebang: true,
		},
		{
			name:         "tacoscript interpreter",
			interpreter:  chshare.Tacoscript,
			partialMatch: true,
			wantCmdStr:   "tacoscript /script.sh",
		},
		{
			name:               "interpreter aliases",
			interpreter:        "taco",
			wantCmdStr:         "/non-standard-interpreter -c /script.sh",
			interpreterAliases: map[string]string{"taco": "/non-standard-interpreter"},
		},
		{
			name:        "interpreter full path",
			interpreter: `/non-standard-interpreter`,
			wantCmdStr:  "/non-standard-interpreter -c /script.sh",
		},
	}
}
