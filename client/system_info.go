package chclient

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"

	"github.com/shirou/gopsutil/host"
)

type CPUInfo struct {
	CPUs     []cpu.InfoStat
	NumCores int
}

type SystemInfo interface {
	Hostname() (string, error)
	HostInfo(context.Context) (*host.InfoStat, error)
	CPUInfo(ctx context.Context) (CPUInfo, error)
	MemoryStats(context.Context) (*mem.VirtualMemoryStat, error)
	Uname(context.Context) (string, error)
	InterfaceAddrs() ([]net.Addr, error)
	GoArch() string
	SystemTime() time.Time
	VirtualizationInfo(ctx context.Context, infoStat *host.InfoStat) (virtSystem, virtRole string, err error)
}

type realSystemInfo struct {
	cmdExec CmdExecutor
}

func NewSystemInfo(cmdExec CmdExecutor) SystemInfo {
	return &realSystemInfo{
		cmdExec: cmdExec,
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

	return cpuInfo, errors.New(strings.Join(errs, ", "))
}

func (s *realSystemInfo) MemoryStats(ctx context.Context) (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemoryWithContext(ctx)
}

func (s *realSystemInfo) SystemTime() time.Time {
	return time.Now()
}

func (s *realSystemInfo) VirtualizationInfo(ctx context.Context, infoStat *host.InfoStat) (virtSystem, virtRole string, err error) {
	if infoStat != nil && infoStat.VirtualizationSystem != "" {
		return strings.ToUpper(infoStat.VirtualizationSystem), strings.ToLower(infoStat.VirtualizationRole), nil
	}

	virtSystem, virtRole, err = s.virtualizationInfo(ctx)
	if err != nil {
		return "", "", err
	}

	return strings.ToUpper(virtSystem), strings.ToLower(virtRole), nil
}
