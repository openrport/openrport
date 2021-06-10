//+build !windows

package script

import (
	"fmt"
	"github.com/cloudradar-monitoring/rport/share/random"
)

func (e *Executor) createClientScriptPath(isPowershell bool) string {
	scriptName := random.UUID4()
	if isPowershell {
		return scriptName + ".ps1"
	}
	return scriptName + ".bat"
}

func (e *Executor) createShell(isPowershell bool) string {
	if isPowershell {
		return "powershell"
	}

	return "cmd"
}

func (e *Executor) createScriptCommand(scriptPath string, isPowerShell bool) string {
	if isPowerShell {
		return fmt.Sprintf("-executionpolicy bypass -file %s", scriptPath)
	}

	return scriptPath
}
