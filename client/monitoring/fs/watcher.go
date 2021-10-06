package fs

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/disk"

	"github.com/cloudradar-monitoring/rport/client/common"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

func DefaultMetrics() []string {
	return []string{"free_b", "total_b"}
}

type FileSystemWatcherConfig struct {
	TypeInclude                 []string
	PathExclude                 []string
	PathExcludeRecurse          bool
	Metrics                     []string
	IdentifyMountpointsByDevice bool
}

type FileSystemWatcher struct {
	AllowedTypes      map[string]struct{}
	ExcludePath       map[string]struct{}
	ExcludedPathCache map[string]bool
	config            *FileSystemWatcherConfig
	prevIOCounters    map[string]*ioCountersMeasurement
	logger            *chshare.Logger
}

func NewWatcher(config FileSystemWatcherConfig, logger *chshare.Logger) *FileSystemWatcher {
	fsWatcher := &FileSystemWatcher{
		AllowedTypes:      map[string]struct{}{},
		ExcludePath:       make(map[string]struct{}),
		ExcludedPathCache: map[string]bool{},
		config:            &config,
		prevIOCounters:    make(map[string]*ioCountersMeasurement),
		logger:            logger,
	}

	for _, t := range config.TypeInclude {
		fsWatcher.AllowedTypes[strings.ToLower(t)] = struct{}{}
	}

	for _, t := range config.PathExclude {
		fsWatcher.ExcludePath[t] = struct{}{}
	}

	return fsWatcher
}

func (fw *FileSystemWatcher) Results() (common.MeasurementsMap, error) {
	results := common.MeasurementsMap{}
	var errs common.ErrorCollector

	partitions, err := getPartitions(fw.config.IdentifyMountpointsByDevice)
	if err != nil {
		errs.Add(fmt.Errorf("[FS] Failed to read partitions:%v", err))
	}

	partitionIOCounters := map[string]*ioUsageInfo{}
	for _, partition := range partitions {
		if _, typeAllowed := fw.AllowedTypes[strings.ToLower(partition.Fstype)]; !typeAllowed {
			fw.logger.Debugf("[FS] fstype excluded: %s", partition.Fstype)
			continue
		}

		pathExcluded := false

		if fw.config.PathExcludeRecurse {
			for path := range fw.ExcludePath {
				if strings.HasPrefix(partition.Mountpoint, path) {
					fw.logger.Debugf("[FS] mountpoint excluded: %s", partition.Mountpoint)
					pathExcluded = true
					break
				}
			}
		}

		if pathExcluded {
			continue
		}

		partitionMountPoint := strings.ToLower(partition.Mountpoint)

		cacheExists := false
		if pathExcluded, cacheExists = fw.ExcludedPathCache[partitionMountPoint]; cacheExists {
			if pathExcluded {
				fw.logger.Debugf("[FS] mountpoint excluded: %s", partition.Fstype)
				continue
			}
		} else {
			pathExcluded = false
			for _, glob := range fw.config.PathExclude {
				pathExcluded, _ = filepath.Match(glob, partition.Mountpoint)
				if pathExcluded {
					break
				}
			}
			fw.ExcludedPathCache[partitionMountPoint] = pathExcluded

			if pathExcluded {
				fw.logger.Debugf("[FS] mountpoint excluded: %s", partition.Mountpoint)
				continue
			}
		}

		usage, err := getFsPartitionUsageInfo(partition.Mountpoint)
		if err != nil {
			errs.Add(fmt.Errorf("[FS] Failed to get usage info for '%s'(%s):%v", partition.Mountpoint, partition.Device, err))
			continue
		}

		fw.fillUsageMetrics(results, partition.Mountpoint, usage)

		ioCounters, err := getPartitionIOCounters(partition.Device)
		if err != nil {
			isNetworkVolumeDrive := partition.Fstype == "smbfs" || partition.Fstype == "nfs"
			if isNetworkVolumeDrive {
				// this info is not available for network shares
				fw.logger.Debugf("[FS] Skipping IO counters for network share '%s' (device %s)", partition.Mountpoint, partition.Device)
				continue
			}

			fw.logger.Errorf("[FS] Failed to get IO counters for '%s' (device %s)", partition.Mountpoint, partition.Device)
			errs.Add(err)
			continue
		}
		currTimestamp := time.Now()
		var prevIOCountersMeasurementTimestamp time.Time
		var prevIOCounters *disk.IOCountersStat
		prevIOCountersMeasurement, prevMeasurementExists := fw.prevIOCounters[partitionMountPoint]
		fw.prevIOCounters[partitionMountPoint] = &ioCountersMeasurement{currTimestamp, ioCounters}
		if prevMeasurementExists {
			prevIOCountersMeasurementTimestamp = prevIOCountersMeasurement.timestamp
			prevIOCounters = prevIOCountersMeasurement.counters

			ioCounters := calcIOCountersUsage(prevIOCounters, ioCounters, currTimestamp.Sub(prevIOCountersMeasurementTimestamp))
			fw.fillIOCounterMetrics(results, partition.Mountpoint, ioCounters)
			partitionIOCounters[partitionMountPoint] = ioCounters
		} else {
			fw.logger.Debugf("[FS] skipping IO usage metrics for %s as it will be available starting from second check", partition.Mountpoint)
		}
	}

	totalIOCounters := calcTotalIOUsage(partitionIOCounters)
	fw.fillTotalIOCountersMetrics(results, totalIOCounters)

	return results, errs.Combine()
}

func (fw *FileSystemWatcher) fillUsageMetrics(results common.MeasurementsMap, mountName string, usage *disk.UsageStat) {
	for _, metric := range fw.config.Metrics {
		resultField := metric + "." + mountName
		switch strings.ToLower(metric) {
		case "free_b":
			results[resultField] = float64(usage.Free)
		case "free_percent":
			results[resultField] = float64(int64((100-usage.UsedPercent)*100+0.5)) / 100
		case "used_percent":
			results[resultField] = float64(int64(usage.UsedPercent*100+0.5)) / 100
		case "total_b":
			results[resultField] = usage.Total
		case "inodes_total":
			results[resultField] = usage.InodesTotal
		case "inodes_free":
			results[resultField] = usage.InodesFree
		case "inodes_used":
			results[resultField] = usage.InodesUsed
		case "inodes_used_percent":
			results[resultField] = float64(int64(usage.InodesUsedPercent*100+0.5)) / 100
		}
	}
}

func (fw *FileSystemWatcher) fillIOCounterMetrics(results common.MeasurementsMap, mountName string, ioCounters *ioUsageInfo) {
	for _, metric := range fw.config.Metrics {
		resultField := metric + "." + mountName
		switch strings.ToLower(metric) {
		case "read_b_per_s":
			results[resultField] = common.RoundToTwoDecimalPlaces(ioCounters.readBytesPerSecond)
		case "write_b_per_s":
			results[resultField] = common.RoundToTwoDecimalPlaces(ioCounters.writeBytesPerSecond)
		case "read_ops_per_s":
			results[resultField] = common.RoundToTwoDecimalPlaces(ioCounters.readOperationsPerSecond)
		case "write_ops_per_s":
			results[resultField] = common.RoundToTwoDecimalPlaces(ioCounters.writeOperationsPerSecond)
		}
	}
}

func (fw *FileSystemWatcher) fillTotalIOCountersMetrics(results common.MeasurementsMap, totalIOCounters *ioUsageInfo) {
	for _, metric := range fw.config.Metrics {
		switch strings.ToLower(metric) {
		case "read_b_per_s":
			results["total_read_B_per_s"] = common.RoundToTwoDecimalPlaces(totalIOCounters.readBytesPerSecond)
		case "write_b_per_s":
			results["total_write_B_per_s"] = common.RoundToTwoDecimalPlaces(totalIOCounters.writeBytesPerSecond)
		case "read_ops_per_s":
			results["total_read_ops_per_s"] = common.RoundToTwoDecimalPlaces(totalIOCounters.readOperationsPerSecond)
		case "write_ops_per_s":
			results["total_write_ops_per_s"] = common.RoundToTwoDecimalPlaces(totalIOCounters.writeOperationsPerSecond)
		}
	}
}
