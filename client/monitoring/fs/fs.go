package fs

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/disk"
)

const fsInfoRequestTimeout = time.Second * 10

type ioCountersMeasurement struct {
	timestamp time.Time
	counters  *disk.IOCountersStat
}

type ioUsageInfo struct {
	readBytesPerSecond       float64
	writeBytesPerSecond      float64
	readOperationsPerSecond  float64
	writeOperationsPerSecond float64
}

func getFsPartitionUsageInfo(mountPoint string) (*disk.UsageStat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fsInfoRequestTimeout)
	defer cancel()
	return disk.UsageWithContext(ctx, mountPoint)
}

func calcIOCountersUsage(prev, curr *disk.IOCountersStat, timeDelta time.Duration) *ioUsageInfo {
	deltaSeconds := timeDelta.Seconds()
	return &ioUsageInfo{
		readBytesPerSecond:       float64(curr.ReadBytes-prev.ReadBytes) / deltaSeconds,
		writeBytesPerSecond:      float64(curr.WriteBytes-prev.WriteBytes) / deltaSeconds,
		readOperationsPerSecond:  float64(curr.ReadCount-prev.ReadCount) / deltaSeconds,
		writeOperationsPerSecond: float64(curr.WriteCount-prev.WriteCount) / deltaSeconds,
	}
}

func calcTotalIOUsage(partitionsUsageInfo map[string]*ioUsageInfo) *ioUsageInfo {
	result := &ioUsageInfo{}
	for _, info := range partitionsUsageInfo {
		result.readBytesPerSecond += info.readBytesPerSecond
		result.writeBytesPerSecond += info.writeBytesPerSecond
		result.readOperationsPerSecond += info.readOperationsPerSecond
		result.writeOperationsPerSecond += info.writeOperationsPerSecond
	}
	return result
}
