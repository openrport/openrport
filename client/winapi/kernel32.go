// +build windows

package winapi

import (
	"fmt"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

var (
	kernel32DLL    = windows.NewLazySystemDLL("kernel32.dll")
	kernel32DLLErr = kernel32DLL.Load()

	// https://docs.microsoft.com/en-us/windows/desktop/api/memoryapi/nf-memoryapi-readprocessmemory
	procReadProcessMemory        = kernel32DLL.NewProc("ReadProcessMemory")
	procReadProcessMemoryLoadErr = procReadProcessMemory.Find()
)

func ReadProcessMemory(processHandle windows.Handle, srcAddr uintptr, dstAddr uintptr, size uintptr) (int, error) {
	if err := checkKernel32ProceduresAvailable(); err != nil {
		return 0, err
	}

	var nBytesRead int
	retCode, _, err := procReadProcessMemory.Call(
		uintptr(processHandle),
		srcAddr,
		dstAddr,
		size,
		uintptr(unsafe.Pointer(&nBytesRead)),
	)

	// ReadProcessMemory function returns 0 in the case of failure
	if retCode == 0 {
		return nBytesRead, fmt.Errorf("winapi call to ReadProcessMemory returned %d. err: %s", retCode, err.Error())
	}

	return nBytesRead, nil
}

func checkKernel32ProceduresAvailable() error {
	if kernel32DLLErr != nil {
		return errors.Wrap(kernel32DLLErr, "winapi: can't load dll kernel32.dll")
	}
	if procReadProcessMemoryLoadErr != nil {
		return errors.Wrap(procReadProcessMemoryLoadErr, "winapi: can't get procedure ReadProcessMemory")
	}
	return nil
}
