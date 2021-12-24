package fs

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/shirou/gopsutil/disk"

	"github.com/cloudradar-monitoring/rport/client/monitoring/helper"
	"github.com/cloudradar-monitoring/rport/share/logger"
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
	logger            *logger.Logger
}

func NewWatcher(config FileSystemWatcherConfig, logger *logger.Logger) *FileSystemWatcher {
	fsWatcher := &FileSystemWatcher{
		AllowedTypes:      map[string]struct{}{},
		ExcludePath:       make(map[string]struct{}),
		ExcludedPathCache: map[string]bool{},
		config:            &config,
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

func (fw *FileSystemWatcher) Results() (helper.MeasurementsMap, error) {
	results := helper.MeasurementsMap{}

	partitions, err := getPartitions(fw.config.IdentifyMountpointsByDevice)
	if err != nil {
		return results, fmt.Errorf("[FS] Failed to read partitions:%v", err)
	}

	var errs error
	for _, partition := range partitions {
		if _, typeAllowed := fw.AllowedTypes[strings.ToLower(partition.Fstype)]; !typeAllowed {
			//fw.logger.Debugf("[FS] fstype excluded: %s", partition.Fstype)
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
			errs = multierror.Append(errs, fmt.Errorf("[FS] Failed to get usage info for '%s'(%s):%v", partition.Mountpoint, partition.Device, err))
			continue
		}

		fw.fillUsageMetrics(results, partition.Mountpoint, usage)
	}

	return results, errs
}

func (fw *FileSystemWatcher) fillUsageMetrics(results helper.MeasurementsMap, mountName string, usage *disk.UsageStat) {
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
