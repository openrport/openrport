// +build !windows,!linux

package fs

import (
	"context"
	"path/filepath"

	"github.com/shirou/gopsutil/disk"
)

func getPartitions(onlyUniqueDevices bool) ([]disk.PartitionStat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fsInfoRequestTimeout)
	defer cancel()

	return disk.PartitionsWithContext(ctx, true)
}

func getPartitionIOCounters(deviceName string) (*disk.IOCountersStat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fsInfoRequestTimeout)
	defer cancel()
	name := filepath.Base(deviceName)
	result, err := disk.IOCountersWithContext(ctx, name)
	if err != nil {
		return nil, err
	}
	ret := result[name]
	return &ret, nil
}
