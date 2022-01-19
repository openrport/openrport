package system

import "strings"

type Interpreter struct {
	InterpreterNameFromInput string
	InterpreterAliases       map[string]string
}

func (i Interpreter) Matches(search string, exact bool) bool {
	if exact {
		if i.InterpreterNameFromInput == "" {
			return search == i.GetDefault()
		}
		return search == i.InterpreterNameFromInput
	}

	resolvedInterpreterStr := i.Get()

	resolvedInterpreterStr = strings.ToLower(resolvedInterpreterStr)

	return search == i.InterpreterNameFromInput || strings.Contains(resolvedInterpreterStr, search)
}
