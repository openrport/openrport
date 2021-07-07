package chclient

import (
	"context"
	"net"
	"time"

	"github.com/shirou/gopsutil/mem"

	"github.com/shirou/gopsutil/host"
)

type mockSystemInfo struct {
	ReturnHostname                string
	ReturnHostnameError           error
	ReturnHostInfo                *host.InfoStat
	ReturnHostInfoError           error
	ReturnCPUInfo                 CPUInfo
	ReturnCPUInfoError            error
	ReturnMemoryStat              *mem.VirtualMemoryStat
	ReturnMemoryError             error
	ReturnUname                   string
	ReturnUnameError              error
	ReturnInterfaceAddrs          []net.Addr
	ReturnInterfaceAddrsError     error
	ReturnGoArch                  string
	ReturnSystemTime              time.Time
	ReturnVirtualizationInfoError error
}

func (s *mockSystemInfo) Hostname() (string, error) {
	return s.ReturnHostname, s.ReturnHostnameError
}

func (s *mockSystemInfo) HostInfo(ctx context.Context) (*host.InfoStat, error) {
	return s.ReturnHostInfo, s.ReturnHostInfoError
}

func (s *mockSystemInfo) Uname(ctx context.Context) (string, error) {
	return s.ReturnUname, s.ReturnUnameError
}

func (s *mockSystemInfo) InterfaceAddrs() ([]net.Addr, error) {
	return s.ReturnInterfaceAddrs, s.ReturnInterfaceAddrsError
}

func (s *mockSystemInfo) GoArch() string {
	return s.ReturnGoArch
}

func (s *mockSystemInfo) CPUInfo(ctx context.Context) (CPUInfo, error) {
	return s.ReturnCPUInfo, s.ReturnCPUInfoError
}

func (s *mockSystemInfo) MemoryStats(ctx context.Context) (*mem.VirtualMemoryStat, error) {
	return s.ReturnMemoryStat, s.ReturnMemoryError
}

func (s *mockSystemInfo) SystemTime() time.Time {
	return s.ReturnSystemTime
}

func (s *mockSystemInfo) VirtualizationInfo(ctx context.Context, infoStat *host.InfoStat) (virtSystem, virtRole string, err error) {
	if infoStat == nil {
		return "", "", s.ReturnVirtualizationInfoError
	}

	return infoStat.VirtualizationSystem, infoStat.VirtualizationRole, s.ReturnVirtualizationInfoError
}
