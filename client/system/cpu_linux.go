// +build linux

package system

import (
	"fmt"
	"math"
	"sync"

	"github.com/shirou/gopsutil/cpu"
)

type lastPercent struct {
	sync.Mutex
	lastCPUTimes    []cpu.TimesStat
	lastPerCPUTimes []cpu.TimesStat
}

var lastCPUPercent lastPercent

// PercentIOWait calculates the percentage of io-wait used either per CPU or combined.
// If an interval of 0 is given it will compare the current cpu times against the last call.
// Returns one value per cpu, or a single value if percpu is set to false.
func PercentIOWait() ([]float64, error) {
	return percentIOWaitFromLastCall(false)
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

func percentIOWaitFromLastCall(percpu bool) ([]float64, error) {
	cpuTimes, err := cpu.Times(percpu)
	if err != nil {
		return nil, err
	}
	lastCPUPercent.Lock()
	defer lastCPUPercent.Unlock()
	var lastTimes []cpu.TimesStat
	if percpu {
		lastTimes = lastCPUPercent.lastPerCPUTimes
		lastCPUPercent.lastPerCPUTimes = cpuTimes
	} else {
		lastTimes = lastCPUPercent.lastCPUTimes
		lastCPUPercent.lastCPUTimes = cpuTimes
	}

	if lastTimes == nil {
		return nil, fmt.Errorf("error getting times for cpu percent. lastTimes was nil")
	}
	return calculateAllIOWait(lastTimes, cpuTimes)
}
