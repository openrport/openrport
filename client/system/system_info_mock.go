package system

import (
	"context"
	"net"
	"time"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type MockSystemInfo struct {
	ReturnHostname                string
	ReturnHostnameError           error
	ReturnHostInfo                *host.InfoStat
	ReturnHostInfoError           error
	ReturnCPUInfo                 CPUInfo
	ReturnCPUInfoError            error
	ReturnCPUPercent              float64
	ReturnCPUPercentError         error
	ReturnCPUPercentIOWait        float64
	ReturnCPUPercentIOWaitError   error
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

func (s *MockSystemInfo) Hostname() (string, error) {
	return s.ReturnHostname, s.ReturnHostnameError
}

func (s *MockSystemInfo) HostInfo(ctx context.Context) (*host.InfoStat, error) {
	return s.ReturnHostInfo, s.ReturnHostInfoError
}

func (s *MockSystemInfo) Uname(ctx context.Context) (string, error) {
	return s.ReturnUname, s.ReturnUnameError
}

func (s *MockSystemInfo) InterfaceAddrs() ([]net.Addr, error) {
	return s.ReturnInterfaceAddrs, s.ReturnInterfaceAddrsError
}

func (s *MockSystemInfo) GoArch() string {
	return s.ReturnGoArch
}

func (s *MockSystemInfo) CPUInfo(ctx context.Context) (CPUInfo, error) {
	return s.ReturnCPUInfo, s.ReturnCPUInfoError
}

func (s *MockSystemInfo) CPUPercent(ctx context.Context) (float64, error) {
	return s.ReturnCPUPercent, s.ReturnCPUPercentError
}

func (s *MockSystemInfo) CPUPercentIOWait(ctx context.Context) (float64, error) {
	return s.ReturnCPUPercentIOWait, s.ReturnCPUPercentIOWaitError
}

func (s *MockSystemInfo) MemoryStats(ctx context.Context) (*mem.VirtualMemoryStat, error) {
	return s.ReturnMemoryStat, s.ReturnMemoryError
}

func (s *MockSystemInfo) SystemTime() time.Time {
	return s.ReturnSystemTime
}

func (s *MockSystemInfo) VirtualizationInfo(ctx context.Context) (virtSystem, virtRole string, err error) {
	return "dummy", "guest", s.ReturnVirtualizationInfoError
}
