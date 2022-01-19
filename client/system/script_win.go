//+build windows

package system

import (
	"os"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func ValidateScriptDirOS(fileInfo os.FileInfo, scriptDir string) error {
	return nil
}

func GetScriptExtensionOS(interpreter Interpreter) string {
	isPowershell := interpreter.Matches(chshare.PowerShell, false)

	if isPowershell {
		return ".ps1"
	}

	return ".bat"
}
