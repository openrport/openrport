//go:build linux
// +build linux

package fs

import (
	"context"

	"github.com/shirou/gopsutil/disk"

	"github.com/cloudradar-monitoring/rport/client/monitoring/helper"
)

func getPartitions(onlyUniqueDevices bool) ([]disk.PartitionStat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fsInfoRequestTimeout)
	defer cancel()

	partitions, err := disk.PartitionsWithContext(ctx, true)
	if err != nil {
		return nil, err
	}

	if !onlyUniqueDevices {
		return partitions, nil
	}

	knownDevices := make([]string, 0)
	filteredPartitions := make([]disk.PartitionStat, 0)
	// the list has the same order of partitions as /proc/self/mountpoints
	// we just pick the first partition, skipping partitions with already-known device name
	for _, p := range partitions {
		if !helper.StrInSlice(p.Device, knownDevices) {
			knownDevices = append(knownDevices, p.Device)
			filteredPartitions = append(filteredPartitions, p)
		}
	}

	return filteredPartitions, nil
}
