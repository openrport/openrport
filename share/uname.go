package chshare

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/host"
)

func Uname() (string, error) {
	if runtime.GOOS == "windows" {
		info, err := host.Info()

		if err != nil {
			return "", err
		}
		return info.Platform + " " + info.PlatformVersion + " " + info.PlatformFamily, nil
	}
	uname, err := exec.LookPath("uname")
	if err != nil {
		return "", err
	}
	b, err := invoke.Command(uname, "-a")
	return strings.TrimSpace(string(b)), err
}
