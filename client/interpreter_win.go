//+build windows

package chclient

import (
	"fmt"
	"path/filepath"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func getInterpreterOs(inpt interpreterProviderInput) (string, error) {
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
