package chclient

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/host"
)

type SystemInfo interface {
	Hostname() (string, error)
	HostInfo(context.Context) (*host.InfoStat, error)
	Uname(context.Context) (string, error)
	InterfaceAddrs() ([]net.Addr, error)
	GoArch() string
}

type realSystemInfo struct {
}

func NewSystemInfo() SystemInfo {
	return &realSystemInfo{}
}

func (s *realSystemInfo) Hostname() (string, error) {
	return os.Hostname()
}

func (s *realSystemInfo) HostInfo(ctx context.Context) (*host.InfoStat, error) {
	return host.InfoWithContext(ctx)
}

func (s *realSystemInfo) Uname(ctx context.Context) (string, error) {
	uname, err := exec.LookPath("uname")
	if err != nil {
		return "", err
	}
	b, err := exec.CommandContext(ctx, uname, "-a").CombinedOutput()
	return strings.TrimSpace(string(b)), err
}

func (s *realSystemInfo) InterfaceAddrs() ([]net.Addr, error) {
	return net.InterfaceAddrs()
}

func (s *realSystemInfo) GoArch() string { return runtime.GOARCH }
