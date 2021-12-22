package chclient

import (
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type interpreterProviderInput struct {
	name       string
	hasShebang bool
	aliasesMap map[string]string
}

type interpreterProvider func(inpt interpreterProviderInput) (string, error)

func getInterpreter(inpt interpreterProviderInput) (string, error) {
	if inpt.aliasesMap != nil {
		if mappedInterpreter, ok := inpt.aliasesMap[inpt.name]; ok {
			return mappedInterpreter, nil
		}
	}

	if inpt.name == chshare.Tacoscript {
		return inpt.name, nil
	}

	return getInterpreterOs(inpt)
}
