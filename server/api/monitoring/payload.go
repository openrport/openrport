package monitoring

type ClientMetricsPayload struct {
	Timestamp             string  `json:"timestamp" db:"timestamp"`
	CPUUsagePercentAvg    float64 `json:"cpu_usage_percent_avg,omitempty" db:"cpu_usage_percent_avg"`
	CPUUsagePercentMin    float64 `json:"cpu_usage_percent_min,omitempty" db:"cpu_usage_percent_min"`
	CPUUsagePercentMax    float64 `json:"cpu_usage_percent_max,omitempty" db:"cpu_usage_percent_max"`
	MemoryUsagePercentAvg float64 `json:"memory_usage_percent_avg,omitempty" db:"memory_usage_percent_avg"`
	MemoryUsagePercentMin float64 `json:"memory_usage_percent_min,omitempty" db:"memory_usage_percent_min"`
	MemoryUsagePercentMax float64 `json:"memory_usage_percent_max,omitempty" db:"memory_usage_percent_max"`
	IoUsagePercentAvg     float64 `json:"io_usage_percent_avg,omitempty" db:"io_usage_percent_avg"`
	IoUsagePercentMin     float64 `json:"io_usage_percent_min,omitempty" db:"io_usage_percent_min"`
	IoUsagePercentMax     float64 `json:"io_usage_percent_max,omitempty" db:"io_usage_percent_max"`
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
		"timestamp":                true,
		"cpu_usage_percent_avg":    true,
		"cpu_usage_percent_min":    true,
		"cpu_usage_percent_max":    true,
		"memory_usage_percent_avg": true,
		"memory_usage_percent_min": true,
		"memory_usage_percent_max": true,
		"io_usage_percent_avg":     true,
		"io_usage_percent_min":     true,
		"io_usage_percent_max":     true,
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
var ClientMetricsFieldsDefault = map[string][]string{"fields[metrics]": {"timestamp", "cpu_usage_percent_avg", "cpu_usage_percent_min", "cpu_usage_percent_max", "memory_usage_percent_avg", "memory_usage_percent_min", "memory_usage_percent_max"}}

var ClientProcessesSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientProcessesFilterDefault = map[string][]string{}
var ClientProcessesFieldsDefault = map[string][]string{"fields[processes]": {"timestamp", "processes"}}

var ClientMountpointsSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientMountpointsFilterDefault = map[string][]string{}
var ClientMountpointsFieldsDefault = map[string][]string{"fields[mountpoints]": {"timestamp", "mountpoints"}}
