//go:build windows
// +build windows

package system

import (
	"os"
	"path/filepath"
	"strings"

	chshare "github.com/realvnc-labs/rport/share"
)

func (i Interpreter) Get() string {
	interpreterNameFromInput := i.InterpreterNameFromInput

	if i.InterpreterAliases != nil && interpreterNameFromInput != "" {
		if mappedInterpreter, ok := i.InterpreterAliases[interpreterNameFromInput]; ok {
			return mappedInterpreter
		}
	}

	if interpreterNameFromInput == "" {
		interpreterNameFromInput = i.GetDefault()
	}

	if interpreterNameFromInput == chshare.CmdShell ||
		interpreterNameFromInput == chshare.Tacoscript ||
		interpreterNameFromInput == chshare.PowerShell {
		interpreterWithAbsPath := i.getInterpreterAbsolutePath(interpreterNameFromInput)

		return interpreterWithAbsPath
	}

	return interpreterNameFromInput
}

func (i Interpreter) getInterpreterAbsolutePath(interpreter string) (absInterpreterPath string) {
	if !strings.HasSuffix(interpreter, ".exe") {
		interpreter += ".exe"
	}

	path := os.Getenv("path")
	for _, dir := range filepath.SplitList(path) {
		absInterpreterPath := filepath.Join(dir, interpreter)
		d, err := os.Stat(absInterpreterPath)
		if err != nil || d.IsDir() {
			continue
		}

		return absInterpreterPath
	}

	return interpreter
}

func (i Interpreter) GetDefault() string {
	return chshare.CmdShell
}
