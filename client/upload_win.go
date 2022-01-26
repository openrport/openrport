//go:build windows
// +build windows

package chclient

var FilePushDenyGlobs = []string{
	`C:\Windows\`, `C:\ProgramData`,
}
