//+build windows

package script

import (
	"github.com/cloudradar-monitoring/rport/share/random"
)

func (e *Executor) createClientScriptPath(isPowershell bool) string {
	scriptName := random.UUID4()

	return scriptName + ".sh"
}

func (e *Executor) createShell(isPowershell bool) string {
	return "sh"
}

func (e *Executor) createScriptCommand(scriptPath string, isPowerShell bool) string {
	return scriptPath
}
