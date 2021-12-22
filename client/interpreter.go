package chclient

import (
	"fmt"
	"path/filepath"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type interpreterProviderInput struct {
	name       string
	os         string
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

	if inpt.os == "windows" {
		return getInterpreterWin(inpt)
	}

	return getInterpreterNix(inpt)
}

func getInterpreterNix(inpt interpreterProviderInput) (string, error) {
	if inpt.hasShebang {
		return "", nil
	}

	if inpt.name != "" && filepath.IsAbs(inpt.name) {
		return inpt.name, nil
	}

	if inpt.name == "" {
		return chshare.UnixShell, nil
	}

	return inpt.name, nil
}

func getInterpreterWin(inpt interpreterProviderInput) (string, error) {
	if inpt.name != "" && filepath.IsAbs(inpt.name) {
		return inpt.name, nil
	}

	if inpt.name == chshare.Tacoscript {
		return inpt.name, nil
	}

	switch inpt.name {
	case "":
		return chshare.CmdShell, nil
	case chshare.CmdShell, chshare.PowerShell:
		return inpt.name, nil
	}

	return "", fmt.Errorf("invalid windows interpreter: %q", inpt.name)
}
