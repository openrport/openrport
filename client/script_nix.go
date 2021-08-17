//+build !windows

package chclient

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

const DefaultScriptDir = "/var/lib/rport/scripts"

func ValidateScriptDirOS(scriptDirSysInfo os.FileInfo, scriptDir string) error {
	isWritable := unix.Access(scriptDir, unix.W_OK) == nil
	if !isWritable {
		return fmt.Errorf("script_dir %s is not writable", scriptDir)
	}

	curUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to fetch current unix user: %w", err)
	}

	dirMode := scriptDirSysInfo.Mode().Perm()
	if dirMode != DefaultDirMode {
		return fmt.Errorf(
			"script_dir %s must be read-writable only by %s[%s]. Change directory mode from 0%o to 0%o. Your setup is insecure",
			scriptDir,
			curUser.Username,
			curUser.Uid,
			dirMode,
			DefaultDirMode,
		)
	}

	unixScriptDirStat, ok := scriptDirSysInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("failed to fetch directory %s owner info", scriptDir)
	}

	scriptDirOwnerUID := strconv.FormatUint(uint64(unixScriptDirStat.Uid), 10)
	if scriptDirOwnerUID != curUser.Uid {
		return fmt.Errorf(
			"script_dir %s must be owned by %s[%s] but it's owned by %s. Your setup is insecure",
			scriptDir,
			curUser.Username,
			curUser.Uid,
			scriptDirOwnerUID,
		)
	}

	return nil
}
