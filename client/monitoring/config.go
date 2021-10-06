package monitoring

import "time"

type Config struct {
	Enabled                       bool          `mapstructure:"enabled"`
	Interval                      time.Duration `mapstructure:"interval"`
	FSTypeInclude                 []string      `mapstructure:"fs_type_include"`
	FSPathExclude                 []string      `mapstructure:"fs_path_exclude"`
	FSPathExcludeRecurse          bool          `mapstructure:"fs_path_exclude_recurse"`
	FSIdentifyMountpointsByDevice bool          `mapstructure:"fs_identify_mountpoints_by_device"`
}
