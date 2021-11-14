package monitoring

import (
	"time"

	"github.com/cloudradar-monitoring/rport/share/types"
)

type CPUUsagePercent struct {
	Avg float64 `json:"avg,omitempty" db:"cpu_usage_percent_avg"`
	Min float64 `json:"min,omitempty" db:"cpu_usage_percent_min"`
	Max float64 `json:"max,omitempty" db:"cpu_usage_percent_max"`
}

type MemoryUsagePercent struct {
	Avg float64 `json:"avg,omitempty" db:"memory_usage_percent_avg"`
	Min float64 `json:"min,omitempty" db:"memory_usage_percent_min"`
	Max float64 `json:"max,omitempty" db:"memory_usage_percent_max"`
}

type IOUsagePercent struct {
	Avg float64 `json:"avg,omitempty" db:"io_usage_percent_avg"`
	Min float64 `json:"min,omitempty" db:"io_usage_percent_min"`
	Max float64 `json:"max,omitempty" db:"io_usage_percent_max"`
}

type ClientGraphMetricsPayload struct {
	Timestamp          time.Time `json:"timestamp,omitempty" db:"timestamp"`
	CPUUsagePercent    `json:"cpu_usage_percent,omitempty"`
	MemoryUsagePercent `json:"memory_usage_percent,omitempty"`
	IOUsagePercent     `json:"io_usage_percent,omitempty"`
}

type ClientMetricsPayload struct {
	Timestamp          time.Time `json:"timestamp,omitempty" db:"timestamp"`
	CPUUsagePercent    float64   `json:"cpu_usage_percent" db:"cpu_usage_percent"`
	MemoryUsagePercent float64   `json:"memory_usage_percent" db:"memory_usage_percent"`
	IOUsagePercent     float64   `json:"io_usage_percent" db:"io_usage_percent"`
}

type ClientProcessesPayload struct {
	Timestamp time.Time        `json:"timestamp" db:"timestamp"`
	Processes types.JSONString `json:"processes" db:"processes"`
}

type ClientMountpointsPayload struct {
	Timestamp   time.Time        `json:"timestamp" db:"timestamp"`
	Mountpoints types.JSONString `json:"mountpoints" db:"mountpoints"`
}

var ClientGraphMetricsSortFields = map[string]bool{
	"timestamp": true,
}

var ClientMetricsSortFields = map[string]bool{
	"timestamp": true,
}

var ClientProcessesSortFields = map[string]bool{
	"timestamp": true,
}

var ClientMountpointsSortFields = map[string]bool{
	"timestamp": true,
}

var ClientMetricsFilterFields = map[string]bool{
	"timestamp[gt]":    true,
	"timestamp[lt]":    true,
	"timestamp[since]": true,
	"timestamp[until]": true,
}

var ClientGraphMetricsFilterFields = map[string]bool{
	"timestamp[gt]":    true,
	"timestamp[lt]":    true,
	"timestamp[since]": true,
	"timestamp[until]": true,
}

var ClientProcessesFilterFields = map[string]bool{
	"timestamp[gt]":    true,
	"timestamp[lt]":    true,
	"timestamp[since]": true,
	"timestamp[until]": true,
}

var ClientMountpointsFilterFields = map[string]bool{
	"timestamp[gt]":    true,
	"timestamp[lt]":    true,
	"timestamp[since]": true,
	"timestamp[until]": true,
}

var ClientGraphMetricsFields = map[string]map[string]bool{
	"graph-metrics": map[string]bool{
		"timestamp":            true,
		"cpu_usage_percent":    true,
		"memory_usage_percent": true,
		"io_usage_percent":     true,
	},
}
var ClientMetricsFields = map[string]map[string]bool{
	"metrics": map[string]bool{
		"timestamp":            true,
		"cpu_usage_percent":    true,
		"memory_usage_percent": true,
		"io_usage_percent":     true,
	},
}

var ClientProcessesFields = map[string]map[string]bool{
	"processes": map[string]bool{
		"timestamp": true,
		"processes": true,
	},
}

var ClientMountpointsFields = map[string]map[string]bool{
	"mountpoints": map[string]bool{
		"timestamp":   true,
		"mountpoints": true,
	},
}

var ClientGraphMetricsSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientGraphMetricsFilterDefault = map[string][]string{}
var ClientGraphMetricsFieldsDefault = map[string][]string{}

var ClientMetricsSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientMetricsFilterDefault = map[string][]string{}
var ClientMetricsFieldsDefault = map[string][]string{"fields[metrics]": {"timestamp", "cpu_usage_percent", "memory_usage_percent", "io_usage_percent"}}

var ClientProcessesSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientProcessesFilterDefault = map[string][]string{}
var ClientProcessesFieldsDefault = map[string][]string{"fields[processes]": {"timestamp", "processes"}}

var ClientMountpointsSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientMountpointsFilterDefault = map[string][]string{}
var ClientMountpointsFieldsDefault = map[string][]string{"fields[mountpoints]": {"timestamp", "mountpoints"}}
