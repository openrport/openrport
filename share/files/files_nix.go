//+build !windows

package files

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"

	"github.com/pkg/errors"
)

func ChangeOwner(path, owner, group string) error {
	if owner == "" && group == "" {
		return nil
	}

	targetUserUID := os.Getuid()
	if owner != "" {
		usr, err := user.Lookup(owner)
		if err != nil {
			return err
		}
		targetUserUID, err = strconv.Atoi(usr.Uid)
		if err != nil {
			return err
		}
	}

	targetGroupGUID := os.Getgid()
	if group != "" {
		gr, err := user.LookupGroup(group)
		if err != nil {
			return err
		}
		targetGroupGUID, err = strconv.Atoi(gr.Gid)
		if err != nil {
			return err
		}
	}

	err := os.Chown(path, targetUserUID, targetGroupGUID)
	if err == nil {
		return nil
	}

	if os.IsPermission(err) {
		return ChangeOwnerExecWithSudo(path, owner, group)
	}

	return err
}

func ChangeOwnerExecWithSudo(path, owner, group string) error {
	if owner == "" && group == "" {
		return nil
	}

	args := []string{
		"sudo",
		"-n",
		"chown",
		fmt.Sprintf("%s:%s", owner, group),
		path,
	}

	cmd := exec.Command(args[0], args[1:]...)

	err := cmd.Run()

	if err != nil {
		return errors.Wrapf(err, "failed to execute %s", cmd.String())
	}

	return nil
}

func Rename(oldPath, newPath string) error {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		return err
	}

	return nil
}
