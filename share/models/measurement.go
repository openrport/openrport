package models

type Measurement struct {
	ClientID           string  `json:"client_id" db:"client_id"`
	Timestamp          int64   `json:"timestamp" db:"timestamp"`
	CPUUsagePercent    float64 `json:"cpu_usage_percent" db:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent" db:"memory_usage_percent"`
	IoUsagePercent     float64 `json:"io_usage_percent" db:"io_usage_percent"`
	Processes          string  `json:"processes" db:"processes"`
	Mountpoints        string  `json:"mountpoints" db:"mountpoints"`
}
