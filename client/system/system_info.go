package system

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/shirou/gopsutil/v3/host"
)

type CPUInfo struct {
	CPUs     []cpu.InfoStat
	NumCores int
}

type SysInfo interface {
	Hostname() (string, error)
	HostInfo(context.Context) (*host.InfoStat, error)
	CPUInfo(ctx context.Context) (CPUInfo, error)
	CPUPercent(ctx context.Context) (float64, error)
	CPUPercentIOWait(ctx context.Context) (float64, error)
	MemoryStats(context.Context) (*mem.VirtualMemoryStat, error)
	Uname(context.Context) (string, error)
	InterfaceAddrs() ([]net.Addr, error)
	GoArch() string
	SystemTime() time.Time
	VirtualizationInfo(ctx context.Context) (virtSystem, virtRole string, err error)
}

type realSystemInfo struct {
	cmdExec     CmdExecutor
	lastCallCPU *LastCallCPU
}

func NewSystemInfo(cmdExec CmdExecutor) SysInfo {
	return &realSystemInfo{
		cmdExec:     cmdExec,
		lastCallCPU: &LastCallCPU{},
	}
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

func (s *realSystemInfo) CPUInfo(ctx context.Context) (CPUInfo, error) {
	cpuInfo := CPUInfo{
		CPUs: []cpu.InfoStat{},
	}
	errs := make([]string, 0, 2)
	cpuInfos, err1 := cpu.InfoWithContext(ctx)
	if err1 == nil {
		cpuInfo.CPUs = cpuInfos
	} else {
		errs = append(errs, err1.Error())
	}

	cpuCount, err2 := cpu.CountsWithContext(ctx, true)
	if err2 == nil {
		cpuInfo.NumCores = cpuCount
	} else {
		errs = append(errs, err2.Error())
	}

	if len(errs) > 0 {
		return cpuInfo, errors.New(strings.Join(errs, ", "))
	}

	return cpuInfo, nil
}

func (s *realSystemInfo) MemoryStats(ctx context.Context) (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemoryWithContext(ctx)
}

func (s *realSystemInfo) CPUPercent(ctx context.Context) (float64, error) {
	percentCPU := 0.0
	percents, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return percentCPU, err
	}

	if len(percents) == 1 {
		percentCPU = percents[0]
	}
	return percentCPU, err
}

func (s *realSystemInfo) CPUPercentIOWait(ctx context.Context) (float64, error) {
	percentIOWait := 0.0
	percents, err := PercentIOWait(s.lastCallCPU)
	if err != nil {
		return percentIOWait, err
	}

	if len(percents) == 1 {
		percentIOWait = percents[0]
	}
	return percentIOWait, err
}

func (s *realSystemInfo) SystemTime() time.Time {
	return time.Now()
}

func (s *realSystemInfo) VirtualizationInfo(ctx context.Context) (virtSystem, virtRole string, err error) {
	virtSystem, virtRole, err = s.virtualizationInfo(ctx)
	if err != nil {
		return "", "", err
	}

	return strings.ToUpper(virtSystem), strings.ToLower(virtRole), nil
}
