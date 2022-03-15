//go:build !windows
// +build !windows

package files

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

func ChangeOwner(path, owner, group string) error {
	if owner == "" && group == "" {
		return nil
	}

	return ChangeOwnerExecWithSudo(path, owner, group)
}

func GetFileUIDAndGID(file string) (uid, gid uint32, err error) {
	fi, err := os.Stat(file)
	if err != nil {
		return 0, 0, err
	}

	if statt, ok := fi.Sys().(*syscall.Stat_t); ok {
		return statt.Uid, statt.Gid, nil
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

	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec

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

	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec

	output, err := cmd.CombinedOutput()

	if err != nil {
		return errors.Wrapf(err, "failed to execute %s: %s", cmd.String(), string(output))
	}

	return nil
}
