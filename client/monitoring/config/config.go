package config

import "time"

const DefaultMonitoringInterval = 60 * time.Second

type MonitoringConfig struct {
	Enabled                       bool          `mapstructure:"enabled"`
	Interval                      time.Duration `mapstructure:"interval"`
	FSTypeInclude                 []string      `mapstructure:"fs_type_include"`
	FSPathExclude                 []string      `mapstructure:"fs_path_exclude"`
	FSPathExcludeRecurse          bool          `mapstructure:"fs_path_exclude_recurse"`
	FSIdentifyMountpointsByDevice bool          `mapstructure:"fs_identify_mountpoints_by_device"`
	PMEnabled                     bool          `mapstructure:"pm_enabled"`
	PMKerneltasksEnabled          bool          `mapstructure:"pm_kerneltasks_enabled"`
	PMMaxNumberProcesses          uint          `mapstructure:"pm_max_number_processes"`
}

func (mc *MonitoringConfig) ParseAndValidate() error {
	if mc.Interval < DefaultMonitoringInterval {
		mc.Interval = DefaultMonitoringInterval
	}

	return nil
}
