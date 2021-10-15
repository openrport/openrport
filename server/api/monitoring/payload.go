package monitoring

import "github.com/cloudradar-monitoring/rport/share/models"

type ClientMetricsPayload struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent,omitempty" db:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent,omitempty" db:"memory_usage_percent"`
	IoUsagePercent     float64 `json:"io_usage_percent,omitempty" db:"io_usage_percent"`
	Processes          string  `json:"processes,omitempty" db:"processes"`
	Mountpoints        string  `json:"mountpoints,omitempty" db:"mountpoints"`
}

var ClientMetricsSortFields = map[string]bool{
	"timestamp": true,
}
var ClientMetricsFilterFields = map[string]bool{
	"timestamp[gt]":    true,
	"timestamp[lt]":    true,
	"timestamp[since]": true,
	"timestamp[until]": true,
}
var ClientMetricsFields = map[string]map[string]bool{
	"metrics": map[string]bool{
		"cpu_usage_percent":    true,
		"memory_usage_percent": true,
		"io_usage_percent":     true,
		"processes":            true,
		"mountpoints":          true,
	},
}

var ClientMetricsSortDefault = map[string][]string{"sort": {"-timestamp"}}
var ClientMetricsFilterDefault = map[string][]string{}
var ClientMetricsFieldsDefault = map[string][]string{"fields[metrics]": {"cpu_usage_percent", "memory_usage_percent"}}

func ConvertToClientMetricsPayload(measurement *models.Measurement, fields map[string]bool) ClientMetricsPayload {
	clientMetricsPayload := ClientMetricsPayload{}
	if _, ok := fields["cpu_usage_percent"]; ok {
		clientMetricsPayload.CPUUsagePercent = measurement.CPUUsagePercent
	}
	if _, ok := fields["memory_usage_percent"]; ok {
		clientMetricsPayload.MemoryUsagePercent = measurement.MemoryUsagePercent
	}
	if _, ok := fields["io_usage_percent"]; ok {
		clientMetricsPayload.IoUsagePercent = measurement.IoUsagePercent
	}
	if _, ok := fields["processes"]; ok {
		clientMetricsPayload.Processes = measurement.Processes
	}
	if _, ok := fields["mountpoints"]; ok {
		clientMetricsPayload.Mountpoints = measurement.Mountpoints
	}

	return clientMetricsPayload
}
