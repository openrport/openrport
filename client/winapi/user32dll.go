// +build windows

package winapi

import (
	"syscall"
	"unsafe"
)

var (
	user32                       = syscall.MustLoadDLL("user32.dll")
	procEnumWindows              = user32.MustFindProc("EnumWindows")
	procGetWindowThreadProcessId = user32.MustFindProc("GetWindowThreadProcessId")
	procIsHungAppWindow          = user32.MustFindProc("IsHungAppWindow")
)

type WindowsEnumerator struct {
	currentList        *map[uint32]syscall.Handle
	compiledCallbackFn uintptr
}

// NewWindowsEnumerator returns an object for enumerating window handles
// it's important not to create too many enumerators as
// there is a runtime limit of how many callback objects can be created
func NewWindowsEnumerator() *WindowsEnumerator {
	list := make(map[uint32]syscall.Handle)
	we := WindowsEnumerator{
		currentList: &list,
	}
	we.compiledCallbackFn = syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
		var windowProcessId uint32
		_, err := GetWindowThreadProcessId(h, &windowProcessId)
		if err != nil {
			// ignore the error
			return 1 // continue enumeration
		}
		list[windowProcessId] = h

		return 1 // continue enumeration
	})
	return &we
}

func (we *WindowsEnumerator) Enumerate() (map[uint32]syscall.Handle, error) {
	*we.currentList = make(map[uint32]syscall.Handle)
	err := EnumWindows(we.compiledCallbackFn, 0)
	if err != nil {
		return nil, err
	}
	return *(we.currentList), nil
}

func EnumWindows(enumFunc uintptr, lparam uintptr) (err error) {
	r1, _, e1 := syscall.Syscall(procEnumWindows.Addr(), 2, uintptr(enumFunc), uintptr(lparam), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func GetWindowThreadProcessId(hwnd syscall.Handle, str *uint32) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetWindowThreadProcessId.Addr(), 2, uintptr(hwnd), uintptr(unsafe.Pointer(str)), 0)
	len = int32(r0)
	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func IsHangWindow(hwnd syscall.Handle) (bool, error) {
	isHang, _, e1 := syscall.Syscall(procIsHungAppWindow.Addr(), 1, uintptr(hwnd), 0, 0)
	if e1 != 0 {
		return false, error(e1)
	}

	return isHang == 1, nil
}
