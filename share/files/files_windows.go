//+build windows
package files

import "os"

func ChangeOwner(path, owner, group string) error {
	return nil
}

func Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
