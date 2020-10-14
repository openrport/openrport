package chshare

import "os"

func IsRunningAsRoot() bool {
	return os.Geteuid() == 0
}
