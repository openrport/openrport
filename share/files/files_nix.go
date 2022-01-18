//go:build !windows
// +build !windows

package files

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"github.com/pkg/errors"
)

func GetUIDByName(name string) (uid int, err error) {
	usr, err := user.Lookup(name)
	if err != nil {
		return 0, err
	}
	uid, err = strconv.Atoi(usr.Uid)
	if err != nil {
		return 0, err
	}

	return uid, nil
}

func GetGidByName(group string) (gid int, err error) {
	gr, err := user.LookupGroup(group)
	if err != nil {
		return 0, err
	}
	gid, err = strconv.Atoi(gr.Gid)
	if err != nil {
		return 0, err
	}

	return gid, nil
}

func ChangeOwner(path, owner, group string) error {
	if owner == "" && group == "" {
		return nil
	}

	var err error
	targetUserUID := os.Getuid()
	if owner != "" {
		targetUserUID, err = GetUIDByName(owner)
		if err != nil {
			return err
		}
	}

	targetGroupGUID := os.Getgid()
	if group != "" {
		targetGroupGUID, err = GetGidByName(group)
		if err != nil {
			return err
		}
	}

	err = os.Chown(path, targetUserUID, targetGroupGUID)
	if err == nil {
		return nil
	}

	if os.IsPermission(err) {
		return ChangeOwnerExecWithSudo(path, owner, group)
	}

	return err
}

func FileOwnerOrGroupMatch(file, owner, group string) (bool, error) {
	fileUID, fileGid, err := GetFileUIDAndGID(file)
	if err != nil {
		return false, errors.Wrapf(err, "failed to read uid and gid of file %s", file)
	}

	if owner != "" {
		ownerUID, err := GetUIDByName(owner)
		if err != nil {
			return false, err
		}

		if fileUID != ownerUID {
			return false, nil
		}
	}

	if group != "" {
		ownerGid, err := GetGidByName(group)
		if err != nil {
			return false, err
		}

		if fileGid != ownerGid {
			return false, nil
		}
	}

	return true, nil
}

func GetFileUIDAndGID(file string) (uid, gid int, err error) {
	fi, err := os.Stat(file)
	if err != nil {
		return 0, 0, err
	}

	if statt, ok := fi.Sys().(*syscall.Stat_t); ok {
		return int(statt.Uid), int(statt.Gid), nil
	}

	return 0, 0, nil
}

func ChangeOwnerExecWithSudo(path, owner, group string) error {
	if owner == "" && group == "" {
		return nil
	}

	chownFullPath, err := exec.LookPath("chown")
	if err != nil {
		return err
	}

	args := []string{
		"sudo",
		"-n",
		chownFullPath,
		fmt.Sprintf("%s:%s", owner, group),
		path,
	}

	cmd := exec.Command(args[0], args[1:]...)

	output, err := cmd.CombinedOutput()

	if err != nil {
		return errors.Wrapf(err, "failed to execute %s: %s", cmd.String(), string(output))
	}

	return nil
}

func Rename(oldPath, newPath string) error {
	err := os.Rename(oldPath, newPath)
	if os.IsPermission(err) {
		return MoveExecWithSudo(oldPath, newPath)
	}
	if err != nil {
		return err
	}

	return nil
}

func MoveExecWithSudo(sourcePath, targetPath string) error {
	mvFullPath, err := exec.LookPath("mv")
	if err != nil {
		return err
	}

	args := []string{
		"sudo",
		"-n",
		mvFullPath,
		sourcePath,
		targetPath,
	}

	cmd := exec.Command(args[0], args[1:]...)

	output, err := cmd.CombinedOutput()

	if err != nil {
		return errors.Wrapf(err, "failed to execute %s: %s", cmd.String(), string(output))
	}

	return nil
}
