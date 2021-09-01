package validation

import "fmt"

const Taco = "taco"
const Powershell = "powershell"
const Cmd = "cmd"

var validInputInterpreter = []string{Cmd, Powershell, Taco}

func ValidateInterpreter(interpreter string, isScript bool) error {
	if interpreter == "" {
		return nil
	}

	if !isScript && interpreter == Taco {
		return fmt.Errorf("%s interpreter can't be used for commands execution", Taco)
	}

	for _, v := range validInputInterpreter {
		if interpreter == v {
			return nil
		}
	}

	return fmt.Errorf("expected interpreter to be one of: %s, actual: %s", validInputInterpreter, interpreter)
}
