package measures

import (
	"time"

	"github.com/realvnc-labs/rport/share/models"
)

type Measure struct {
	UID                string           `json:"uid"` // unique id for idempotency
	ClientID           string           `json:"client_id"`
	Timestamp          time.Time        `json:"timestamp"`
	CPUUsagePercent    float64          `json:"cpu_usage_percent"`
	MemoryUsagePercent float64          `json:"memory_usage_percent"`
	IoUsagePercent     float64          `json:"io_usage_percent"`
	NetLan             *models.NetBytes `json:"netlan"`
	NetWan             *models.NetBytes `json:"netwan"`

	Processes   []Process    `json:"processes"`
	MountPoints []MountPoint `json:"mountpoints"`
}

type NetBytes struct {
	In  int `json:"in"`
	Out int `json:"out"`
}

type Process struct {
	Name    string `json:"name"`
	CmdLine string `json:"cmdline"`
}

type MountPoint struct {
	Name       string `json:"name"`
	FreeBytes  uint64 `json:"free_b"`
	TotalBytes uint64 `json:"total_b"`
}

type Measures []*Measure
