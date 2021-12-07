package models

import (
	"time"
)

type NetBytes struct {
	In  int `json:"in"`
	Out int `json:"out"`
}

type Measurement struct {
	ClientID           string    `json:"client_id" db:"client_id"`
	Timestamp          time.Time `json:"timestamp" db:"timestamp"`
	CPUUsagePercent    float64   `json:"cpu_usage_percent" db:"cpu_usage_percent"`
	MemoryUsagePercent float64   `json:"memory_usage_percent" db:"memory_usage_percent"`
	IoUsagePercent     float64   `json:"io_usage_percent" db:"io_usage_percent"`
	Processes          string    `json:"processes" db:"processes"`
	Mountpoints        string    `json:"mountpoints" db:"mountpoints"`
	NetLan             *NetBytes `json:"net_lan" db:"net_lan"`
	NetWan             *NetBytes `json:"net_wan" db:"net_wan"`
}
