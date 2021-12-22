//+build !windows

package chclient

import (
	"path/filepath"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func getInterpreterOs(inpt interpreterProviderInput) (string, error) {
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
