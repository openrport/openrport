//+build windows

package system

import (
	"os"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func ValidateScriptDirOS(fileInfo os.FileInfo, scriptDir string) error {
	return nil
}

func GetScriptExtensionOS(interpreter string) string {
	if interpreter == chshare.PowerShell {
		return ".ps1"
	}

	return ".bat"
}
