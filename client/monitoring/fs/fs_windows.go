// +build windows

package fs

import (
	"bytes"
	"strings"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/cloudradar-monitoring/cagent/pkg/winapi"
)

const (
	minDriveLabelChar = 65
	maxDriveLabelChar = 90

	driveTypeRemovable = 2
	driveTypeFixed     = 3
	driveTypeRemote    = 4
	driveTypeCDROM     = 5
)

// getPartitions is an improved version of gopsutil/disk.GetPartitions()
// which is capable of determining the fs type of network drives.
func getPartitions(onlyUniqueDevices bool) ([]disk.PartitionStat, error) {
	var result []disk.PartitionStat
	lpBuffer := make([]byte, 254)
	bufferPtr := unsafe.Pointer(&lpBuffer[0])
	diskret, err := windows.GetLogicalDriveStrings(uint32(len(lpBuffer)), (*uint16)(bufferPtr))
	if diskret == 0 {
		return result, err
	}

	for _, v := range lpBuffer {
		if v >= minDriveLabelChar && v <= maxDriveLabelChar {
			path := string(v) + ":"
			typepath, _ := windows.UTF16PtrFromString(path)
			typeret := windows.GetDriveType(typepath)
			if typeret == 0 {
				return result, windows.GetLastError()
			}

			if typeret == driveTypeRemovable || typeret == driveTypeFixed || typeret == driveTypeRemote || typeret == driveTypeCDROM {
				lpVolumeNameBuffer := make([]byte, 256)
				lpVolumeSerialNumber := int64(0)
				lpMaximumComponentLength := int64(0)
				lpFileSystemFlags := int64(0)
				lpFileSystemNameBuffer := make([]byte, 256)
				volpath, _ := windows.UTF16PtrFromString(string(v) + ":/")

				err := windows.GetVolumeInformation(
					volpath,
					(*uint16)(unsafe.Pointer(&lpVolumeNameBuffer[0])),
					uint32(len(lpVolumeNameBuffer)),
					(*uint32)(unsafe.Pointer(&lpVolumeSerialNumber)),
					(*uint32)(unsafe.Pointer(&lpMaximumComponentLength)),
					(*uint32)(unsafe.Pointer(&lpFileSystemFlags)),
					(*uint16)(unsafe.Pointer(&lpFileSystemNameBuffer[0])),
					uint32(len(lpFileSystemNameBuffer)),
				)
				if err != nil {
					if typeret == driveTypeCDROM || typeret == driveTypeRemovable {
						continue // device is not ready will happen if there is no disk in the drive
					}
					return result, err
				}
				opts := "rw"
				if lpFileSystemFlags&disk.FileReadOnlyVolume != 0 {
					opts = "ro"
				}
				if lpFileSystemFlags&disk.FileFileCompression != 0 {
					opts += ".compress"
				}

				fsType := string(bytes.Replace(lpFileSystemNameBuffer, []byte("\x00"), []byte(""), -1))
				if typeret == driveTypeRemote {
					remoteDriveType, err := tryRetrieveRemoteDriveFSType(typepath)
					if err != nil {
						return result, err
					}
					if remoteDriveType != "" {
						fsType = remoteDriveType
					}
				}

				d := disk.PartitionStat{
					Mountpoint: path,
					Device:     path,
					Fstype:     fsType,
					Opts:       opts,
				}
				result = append(result, d)
			}
		}
	}
	return result, nil
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

func getPartitionIOCounters(deviceName string) (*disk.IOCountersStat, error) {
	if err := enablePerformanceCounters(); err != nil {
		return nil, err
	}

	var uncPath = `\\.\` + deviceName
	diskPerformance, err := winapi.GetDiskPerformance(uncPath)
	if err != nil {
		return nil, err
	}
	return &disk.IOCountersStat{
		Name:       deviceName,
		ReadCount:  uint64(diskPerformance.ReadCount),
		WriteCount: uint64(diskPerformance.WriteCount),
		ReadBytes:  uint64(diskPerformance.BytesRead),
		WriteBytes: uint64(diskPerformance.BytesWritten),
		ReadTime:   uint64(diskPerformance.ReadTime),
		WriteTime:  uint64(diskPerformance.WriteTime),
	}, nil
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
