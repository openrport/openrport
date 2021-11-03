package monitoring

type CPUUsagePercent struct {
	Value float64 `json:"value,omitempty" db:"cpu_usage_percent"`
	Min   float64 `json:"min,omitempty" db:"cpu_usage_percent_min"`
	Max   float64 `json:"max,omitempty" db:"cpu_usage_percent_max"`
}

type MemoryUsagePercent struct {
	Value float64 `json:"value,omitempty" db:"memory_usage_percent"`
	Min   float64 `json:"min,omitempty" db:"memory_usage_percent_min"`
	Max   float64 `json:"max,omitempty" db:"memory_usage_percent_max"`
}

type IOUsagePercent struct {
	Value float64 `json:"value,omitempty" db:"io_usage_percent"`
	Min   float64 `json:"min,omitempty" db:"io_usage_percent_min"`
	Max   float64 `json:"max,omitempty" db:"io_usage_percent_max"`
}

type ClientMetricsPayload struct {
	Timestamp          string `json:"timestamp,omitempty" db:"timestamp"`
	CPUUsagePercent    `json:"cpu_usage_percent,omitempty"`
	MemoryUsagePercent `json:"memory_usage_percent,omitempty"`
	IOUsagePercent     `json:"io_usage_percent,omitempty"`
}

type ClientProcessesPayload struct {
	Timestamp string `json:"timestamp" db:"timestamp"`
	Processes string `json:"processes" db:"processes"`
}

type ClientMountpointsPayload struct {
	Timestamp   string `json:"timestamp" db:"timestamp"`
	Mountpoints string `json:"mountpoints" db:"mountpoints"`
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
	"limit":            true,
}

var ClientProcessesFilterFields = map[string]bool{
	"timestamp": true,
}

var ClientMountpointsFilterFields = map[string]bool{
	"timestamp": true,
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

var ClientMetricsSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientMetricsFilterDefault = map[string][]string{}
var ClientMetricsFieldsDefault = map[string][]string{"fields[metrics]": {"timestamp", "cpu_usage_percent", "memory_usage_percent"}}

var ClientProcessesSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientProcessesFilterDefault = map[string][]string{}
var ClientProcessesFieldsDefault = map[string][]string{"fields[processes]": {"timestamp", "processes"}}

var ClientMountpointsSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientMountpointsFilterDefault = map[string][]string{}
var ClientMountpointsFieldsDefault = map[string][]string{"fields[mountpoints]": {"timestamp", "mountpoints"}}
