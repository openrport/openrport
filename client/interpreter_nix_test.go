//+build !windows

package chclient

import (
	chshare "github.com/cloudradar-monitoring/rport/share"
)

func getInterpreterTestCases() []interpreterTestCase {
	return []interpreterTestCase{
		{
			name:            "empty",
			interpreter:     "",
			wantInterpreter: chshare.UnixShell,
			wantErrContains: "",
		},
		{
			name:            "non empty",
			interpreter:     chshare.UnixShell,
			wantInterpreter: chshare.UnixShell,
		},
		{
			name:            "empty interpreter",
			interpreter:     "",
			wantInterpreter: chshare.UnixShell,
			wantErrContains: "",
		},
		{
			name:            "hasShebang, interpreter empty",
			wantInterpreter: "",
			boolHasShebang:  true,
		},
		{
			name:            "hasShebang, interpreter not empty",
			interpreter:     chshare.UnixShell,
			wantInterpreter: "",
			boolHasShebang:  true,
		},
		{
			name:            "tacoscript interpreter",
			interpreter:     chshare.Tacoscript,
			wantInterpreter: chshare.Tacoscript,
		},
		{
			name:               "interpreter aliases",
			interpreter:        "taco",
			wantInterpreter:    chshare.Tacoscript,
			interpreterAliases: map[string]string{"taco": chshare.Tacoscript},
		},
		{
			name:            "interpreter full path",
			interpreter:     `/usr/local/bin/bash`,
			wantInterpreter: `/usr/local/bin/bash`,
		},
	}
}
