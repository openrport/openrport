//go:build !windows
// +build !windows

package chclient

var FileReceptionGlobs = []string{
	"/bin", "/sbin", "/boot", "/usr/bin", "/usr/sbin", "/dev", "/lib*", "/run",
}
