//go:build !windows
// +build !windows

package system

import (
	"context"
	"io/ioutil"
	"log"
	"os"
)

const devicesInfoPath = "/proc/bus/pci/devices"

var vDevices = map[string]string{
	"/dev/vmbus/hv_vss": VirtualSystemHyperV,
	"/dev/vmbus/hv_kvp": VirtualSystemHyperV,
}

func (s *realSystemInfo) virtualizationInfo(ctx context.Context) (virtSystem, virtRole string, err error) {
	for i, v := range vDevices {
		if _, err := os.Stat(i); err == nil {
			return v, VirtualSystemRoleGuest, nil
		}
	}
	_, err = os.Stat(devicesInfoPath)
	log.Println("Looking for existence ", devicesInfoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}

		return "", "", err
	}

	fileContent, err := ioutil.ReadFile(devicesInfoPath)
	if err != nil {
		return "", "", err
	}

	log.Println("Looking at ", devicesInfoPath, string(fileContent))
	virtSystem, virtRole = getVirtInfoFromNixDevicesList(string(fileContent))

	return virtSystem, virtRole, nil
}
