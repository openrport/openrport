package system

import (
	"fmt"
	"time"
)

const (
	UnixShell  = "/bin/sh"
	CmdShell   = "cmd"
	PowerShell = "powershell"
)

// GetInterpreter var is used to override in tests
var GetInterpreter = func(inputInterpreter, os string, hasShebang bool) (string, error) {
	if os == "windows" {
		switch inputInterpreter {
		case "":
			return CmdShell, nil
		case CmdShell, PowerShell:
			return inputInterpreter, nil
		}
		return "", fmt.Errorf("invalid windows command interpreter: %q", inputInterpreter)
	}

	if hasShebang {
		return "", nil
	}

	if inputInterpreter != "" {
		return "", fmt.Errorf("for unix clients a command interpreter should not be specified, got: %q", inputInterpreter)
	}
	return UnixShell, nil
}

// Now is used to stub time.Now in tests
var Now = time.Now
