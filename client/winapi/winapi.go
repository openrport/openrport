// +build windows

package winapi

import (
	"unsafe"

	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField("package", "winapi")

	procGlobalFree = kernel32DLL.NewProc("GlobalFree")
)

func add(p unsafe.Pointer, x uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

func GlobalFree(mem *uint16) error {
	if mem == nil {
		return nil
	}
	if err := procGlobalFree.Find(); err != nil {
		return err
	}
	r, _, err := procGlobalFree.Call(uintptr(unsafe.Pointer(mem)))
	if r == 0 {
		return nil
	}

	return err
}
