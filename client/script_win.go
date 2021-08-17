//+build windows

package chclient

import "os"

const DefaultScriptDir = "C:\\Program Files\\rport\\scripts"

func ValidateScriptDirOS(fileInfo os.FileInfo, scriptDir string) error {
	return nil
}
