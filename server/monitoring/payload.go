package monitoring

import (
	"time"

	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/types"
)

const (
	LinkCPUPercent    = "cpu_usage_percent"
	LinkMemPercent    = "mem_usage_percent"
	LinkIOPercent     = "io_usage_percent"
	LinkNetPercentLan = "net_usage_percent_lan"
	LinkNetBPSLan     = "net_usage_bps_lan"
	LinkNetPercentWan = "net_usage_percent_wan"
	LinkNetBPSWan     = "net_usage_bps_wan"
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

type NetUsagePercentLan struct {
	InAvg  *float64 `json:"in_avg,omitempty" db:"net_usage_percent_lan_in_avg"`
	InMin  *float64 `json:"in_min,omitempty" db:"net_usage_percent_lan_in_min"`
	InMax  *float64 `json:"in_max,omitempty" db:"net_usage_percent_lan_in_max"`
	OutAvg *float64 `json:"out_avg,omitempty" db:"net_usage_percent_lan_out_avg"`
	OutMin *float64 `json:"out_min,omitempty" db:"net_usage_percent_lan_out_min"`
	OutMax *float64 `json:"out_max,omitempty" db:"net_usage_percent_lan_out_max"`
}

type NetUsagePercentWan struct {
	InAvg  *float64 `json:"in_avg,omitempty" db:"net_usage_percent_wan_in_avg"`
	InMin  *float64 `json:"in_min,omitempty" db:"net_usage_percent_wan_in_min"`
	InMax  *float64 `json:"in_max,omitempty" db:"net_usage_percent_wan_in_max"`
	OutAvg *float64 `json:"out_avg,omitempty" db:"net_usage_percent_wan_out_avg"`
	OutMin *float64 `json:"out_min,omitempty" db:"net_usage_percent_wan_out_min"`
	OutMax *float64 `json:"out_max,omitempty" db:"net_usage_percent_wan_out_max"`
}

type NetUsageBPSLan struct {
	InAvg  *float64 `json:"in_avg,omitempty" db:"net_usage_bps_lan_in_avg"`
	InMin  *float64 `json:"in_min,omitempty" db:"net_usage_bps_lan_in_min"`
	InMax  *float64 `json:"in_max,omitempty" db:"net_usage_bps_lan_in_max"`
	OutAvg *float64 `json:"out_avg,omitempty" db:"net_usage_bps_lan_out_avg"`
	OutMin *float64 `json:"out_min,omitempty" db:"net_usage_bps_lan_out_min"`
	OutMax *float64 `json:"out_max,omitempty" db:"net_usage_bps_lan_out_max"`
}

type NetUsageBPSWan struct {
	InAvg  *float64 `json:"in_avg,omitempty" db:"net_usage_bps_wan_in_avg"`
	InMin  *float64 `json:"in_min,omitempty" db:"net_usage_bps_wan_in_min"`
	InMax  *float64 `json:"in_max,omitempty" db:"net_usage_bps_wan_in_max"`
	OutAvg *float64 `json:"out_avg,omitempty" db:"net_usage_bps_wan_out_avg"`
	OutMin *float64 `json:"out_min,omitempty" db:"net_usage_bps_wan_out_min"`
	OutMax *float64 `json:"out_max,omitempty" db:"net_usage_bps_wan_out_max"`
}

type ClientGraphMetricsPayload struct {
	Timestamp          time.Time `json:"timestamp,omitempty" db:"timestamp"`
	CPUUsagePercent    `json:"cpu_usage_percent,omitempty"`
	MemoryUsagePercent `json:"memory_usage_percent,omitempty"`
	IOUsagePercent     `json:"io_usage_percent,omitempty"`
}

type ClientGraphMetricsGraphPayload struct {
	Timestamp           time.Time `json:"timestamp,omitempty" db:"timestamp"`
	*CPUUsagePercent    `json:"cpu_usage_percent,omitempty"`
	*MemoryUsagePercent `json:"memory_usage_percent,omitempty"`
	*IOUsagePercent     `json:"io_usage_percent,omitempty"`
	*NetUsagePercentLan `json:"net_usage_percent_lan,omitempty"`
	*NetUsagePercentWan `json:"net_usage_percent_wan,omitempty"`
	*NetUsageBPSLan     `json:"net_usage_bps_lan,omitempty"`
	*NetUsageBPSWan     `json:"net_usage_bps_wan,omitempty"`
}

var ClientGraphNameToField = map[string]string{
	"cpu_usage_percent":     "cpu_usage_percent",
	"mem_usage_percent":     "memory_usage_percent",
	"io_usage_percent":      "io_usage_percent",
	"net_usage_percent_lan": "net_lan_in",
	"net_usage_percent_wan": "net_wan_in",
	"net_usage_bps_lan":     "net_lan_in",
	"net_usage_bps_wan":     "net_wan_in",
}

var ClientGraphNameToAlias = map[string]string{
	"cpu_usage_percent":     "cpu_usage_percent",
	"mem_usage_percent":     "memory_usage_percent",
	"io_usage_percent":      "io_usage_percent",
	"net_usage_percent_lan": "net_usage_percent_lan_in",
	"net_usage_percent_wan": "net_usage_percent_wan_in",
	"net_usage_bps_lan":     "net_usage_bps_lan_in",
	"net_usage_bps_wan":     "net_usage_bps_wan_in",
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

type GraphMetricsLinksPayload struct {
	CPUUsagePercent    *string `json:"cpu_usage_percent,omitempty"`
	MemUsagePercent    *string `json:"mem_usage_percent,omitempty"`
	IOUsagePercent     *string `json:"io_usage_percent,omitempty"`
	NetLanUsagePercent *string `json:"net_usage_percent_lan,omitempty"`
	NetWanUsagePercent *string `json:"net_usage_percent_wan,omitempty"`
	NetLanUsageBPS     *string `json:"net_usage_bps_lan,omitempty"`
	NetWanUsageBPS     *string `json:"net_usage_bps_wan,omitempty"`
}

func NewGraphMetricsLink(requestInfo *query.RequestInfo, target string) *string {
	link := requestInfo.URL + "/" + target
	return &link
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
