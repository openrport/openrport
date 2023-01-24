//go:build windows
// +build windows

package fs

import (
	"bytes"
	"context"
	"strings"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// getPartitions wraps gopsutil/disk.Partitions and also determines fs type of network drives
func getPartitions(onlyUniqueDevices bool) ([]disk.PartitionStat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fsInfoRequestTimeout)
	defer cancel()

	partitions, err := disk.PartitionsWithContext(ctx, true)
	if err != nil {
		// Ignore warnings. Warning might happen if the network drive is not connected
		if _, ok := err.(*disk.Warnings); !ok {
			return nil, err
		}
	}
	for i, partition := range partitions {
		typepath, _ := windows.UTF16PtrFromString(partition.Mountpoint)
		remoteDriveType, err := tryRetrieveRemoteDriveFSType(typepath)
		// Ignore errors that might happen trying to identify remote drive type
		if err != nil {
			continue
		}
		if remoteDriveType != "" {
			partitions[i].Fstype = remoteDriveType
		}
	}
	return partitions, nil
}

// tryRetrieveRemoteDriveFSType can detect the original network share filesystem.
// If filesystem wasn't recognized, the empty string returned.
// Based on some insights from cygwin implementation.
func tryRetrieveRemoteDriveFSType(drivePath *uint16) (string, error) {
	lpTargetBuffer := make([]byte, 256)
	_, err := windows.QueryDosDevice(drivePath, (*uint16)(unsafe.Pointer(&lpTargetBuffer[0])), uint32(len(lpTargetBuffer)))
	if err != nil {
		return "", errors.Wrapf(err, "while QueryDosDevice call")
	}

	dosDeviceName := string(bytes.Replace(lpTargetBuffer, []byte("\x00"), []byte(""), -1))
	if strings.Contains(dosDeviceName, "LanmanRedirector\\") {
		return "smbfs", nil
	}

	if strings.Contains(dosDeviceName, "MRxNfs\\") {
		return "nfs", nil
	}

	return "", nil
}

// enablePerformanceCounters will enable performance counters by adding the EnableCounterForIoctl registry key
func enablePerformanceCounters() error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, "SYSTEM\\CurrentControlSet\\Services\\partmgr", registry.READ|registry.WRITE)
	if err != nil {
		return errors.Errorf("cannot open new key in the registry in order to enable the performance counters: %s", err)
	}
	val, _, err := key.GetIntegerValue("EnableCounterForIoctl")
	if val != 1 || err != nil {
		if err = key.SetDWordValue("EnableCounterForIoctl", 1); err != nil {
			return errors.Errorf("cannot create HKLM:SYSTEM\\CurrentControlSet\\Services\\Partmgr\\EnableCounterForIoctl key in the registry in order to enable the performance counters: %s", err)
		}
		logrus.Info("The registry key EnableCounterForIoctl at HKLM:SYSTEM\\CurrentControlSet\\Services\\Partmgr has been created in order to enable the performance counters")
	}
	return nil
}
