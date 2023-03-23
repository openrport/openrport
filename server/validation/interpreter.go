package validation

import (
	"fmt"

	chshare "github.com/realvnc-labs/rport/share"
)

var validInputInterpreter = []string{chshare.CmdShell, chshare.PowerShell, chshare.Tacoscript}

func ValidateInterpreter(interpreter string, isScript bool) error {
	// we skip validation for scripts because server is not able to detect invalid values as user might use
	// interpreter aliases or full paths which are not accessible on the server
	if interpreter == "" || isScript {
		return nil
	}

	if interpreter == chshare.Tacoscript {
		return fmt.Errorf("%s interpreter can't be used for commands execution", chshare.Tacoscript)
	}

	for _, v := range validInputInterpreter {
		if interpreter == v {
			return nil
		}
	}

	return fmt.Errorf("expected interpreter to be one of: %s, actual: %s", validInputInterpreter, interpreter)
}
