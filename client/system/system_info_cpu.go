package system

import (
	"fmt"
	"math"
	"sync"

	"github.com/shirou/gopsutil/v3/cpu"
)

type LastCallCPU struct {
	sync.Mutex
	lastCPUTimes    []cpu.TimesStat
	lastPerCPUTimes []cpu.TimesStat
}

// PercentIOWait calculates the percentage of io-wait combined for all CPU's.
// The current cpu times are compared against cpu times from the last call.
func PercentIOWait(lastCall *LastCallCPU) ([]float64, error) {
	return percentIOWaitFromLastCall(lastCall, false)
}

func getAllIOWait(t cpu.TimesStat) (float64, float64) {
	all := t.User + t.System + t.Nice + t.Iowait + t.Irq +
		t.Softirq + t.Steal
	return all, t.Iowait
}

func calculateIOWait(t1, t2 cpu.TimesStat) float64 {
	t1All, t1IOWait := getAllIOWait(t1)
	t2All, t2IOWait := getAllIOWait(t2)

	if t2IOWait <= t1IOWait {
		return 0
	}
	return math.Min(100, math.Max(0, (t2IOWait-t1IOWait)/(t2All-t1All)*100))
}

func calculateAllIOWait(t1, t2 []cpu.TimesStat) ([]float64, error) {
	// Make sure the CPU measurements have the same length.
	if len(t1) != len(t2) {
		return nil, fmt.Errorf(
			"received two CPU counts: %d != %d",
			len(t1), len(t2),
		)
	}

	ret := make([]float64, len(t1))
	for i, t := range t2 {
		ret[i] = calculateIOWait(t1[i], t)
	}
	return ret, nil
}

func percentIOWaitFromLastCall(lastCall *LastCallCPU, percpu bool) ([]float64, error) {
	cpuTimes, err := cpu.Times(percpu)
	if err != nil {
		return nil, err
	}
	lastCall.Lock()
	defer lastCall.Unlock()
	var lastTimes []cpu.TimesStat
	if percpu {
		lastTimes = lastCall.lastPerCPUTimes
		lastCall.lastPerCPUTimes = cpuTimes
	} else {
		lastTimes = lastCall.lastCPUTimes
		lastCall.lastCPUTimes = cpuTimes
	}

	if lastTimes == nil {
		return nil, fmt.Errorf("error getting times for cpu percent. lastTimes was nil")
	}
	return calculateAllIOWait(lastTimes, cpuTimes)
}
