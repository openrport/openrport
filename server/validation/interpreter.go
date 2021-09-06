package validation

import (
	"fmt"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

var validInputInterpreter = []string{chshare.CmdShell, chshare.PowerShell, chshare.Taco}

func ValidateInterpreter(interpreter string, isScript bool) error {
	if interpreter == "" {
		return nil
	}

	if !isScript && interpreter == chshare.Taco {
		return fmt.Errorf("%s interpreter can't be used for commands execution", chshare.Taco)
	}

	for _, v := range validInputInterpreter {
		if interpreter == v {
			return nil
		}
	}

	return fmt.Errorf("expected interpreter to be one of: %s, actual: %s", validInputInterpreter, interpreter)
}
