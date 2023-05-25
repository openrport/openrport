package clientupdates

import (
	"time"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/measures"
)

type Client struct {
	Timestamp time.Time `json:"timestamp"`
	UID       string    // unique id for idempotency

	Version string `json:"version"`
	ID      string `json:"id"`
	Name    string `json:"name"`

	Address         string     `json:"address"`
	DisconnectedAt  *time.Time `json:"disconnected_at"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at"`
	ConnectionState string     `json:"connection_state"`

	Tags   []string          `json:"tags"`
	Labels map[string]string `json:"labels"`

	UpdatesAvailable         int `json:"updates_available"`
	SecurityUpdatesAvailable int `json:"security_updates_available"`

	OS                     string   `json:"os"`
	OSArch                 string   `json:"os_arch"`
	OSFamily               string   `json:"os_family"`
	OSKernel               string   `json:"os_kernel"`
	OSFullName             string   `json:"os_full_name"`
	OSVersion              string   `json:"os_version"`
	OSVirtualizationSystem string   `json:"os_virtualization_system"`
	OSVirtualizationRole   string   `json:"os_virtualization_role"`
	NumCPUs                int      `json:"num_cpus"`
	MemoryTotal            uint64   `json:"mem_total"`
	Timezone               string   `json:"timezone"`
	Hostname               string   `json:"hostname"`
	IPv4                   []string `json:"ipv4"`
	IPv6                   []string `json:"ipv6"`

	Measures measures.Measures `json:"-"`
}

type Clients []*Client
