//go:build !windows
// +build !windows

package system

import (
	"context"
	"io/ioutil"
	"os"
)

const devicesInfoPath = "/proc/bus/pci/devices"

func (s *realSystemInfo) virtualizationInfo(ctx context.Context) (virtSystem, virtRole string, err error) {
	_, err = os.Stat(devicesInfoPath)
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

	virtSystem, virtRole = getVirtInfoFromNixDevicesList(string(fileContent))

	return virtSystem, virtRole, nil
}
