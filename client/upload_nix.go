//+build !windows

package chclient

var FilePushDenyGlobs = []string{
	"/bin", "/sbin", "/boot", "/usr/bin", "/usr/sbin", "/dev", "/lib*", "/run",
}
