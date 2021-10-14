package processes

import (
	"encoding/json"
	"sort"

	"github.com/shirou/gopsutil/mem"

	"github.com/cloudradar-monitoring/rport/client/monitoring/config"
	"github.com/cloudradar-monitoring/rport/client/monitoring/docker"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ProcessHandler struct {
	config        config.MonitoringConfig
	logger        *chshare.Logger
	dockerHandler *docker.Handler
}

func NewProcessHandler(config config.MonitoringConfig, logger *chshare.Logger, dockerHandler *docker.Handler) *ProcessHandler {
	return &ProcessHandler{config: config, logger: logger, dockerHandler: dockerHandler}
}

type ProcStat struct {
	PID                    int     `json:"pid"`
	ParentPID              int     `json:"parent_pid"`
	ProcessGID             int     `json:"-"`
	Name                   string  `json:"name"`
	Cmdline                string  `json:"cmdline"`
	State                  string  `json:"state"`
	Container              string  `json:"container,omitempty"`
	CPUAverageUsagePercent float32 `json:"cpu_avg_usage_percent,omitempty"`
	RSS                    uint64  `json:"rss"` // Resident Set Size
	VMS                    uint64  `json:"vms"` // Virtual Memory Size
	MemoryUsagePercent     float32 `json:"memory_usage_percent"`
}

func (ph *ProcessHandler) GetProcessesJSON(memStat *mem.VirtualMemoryStat) (string, error) {
	if !ph.config.Enabled {
		return "", nil
	}
	var systemMemorySize uint64
	if memStat == nil {
		ph.logger.Debugf("System memory information is unavailable. Some process stats will not be calculated...")
	} else {
		systemMemorySize = memStat.Total
	}
	procs, err := ph.processes(systemMemorySize)
	if err != nil {
		ph.logger.Errorf(err.Error())
		return "", err
	}

	return toJSON(filterProcs(procs, &ph.config)), nil
}

func filterProcs(procs []*ProcStat, cfg *config.MonitoringConfig) []*ProcStat {
	// sort by PID descending:
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].PID > procs[j].PID
	})

	result := make([]*ProcStat, 0, cfg.PMMaxNumberProcesses)
	var count uint
	for _, p := range procs {
		if count == cfg.PMMaxNumberProcesses {
			break
		}

		if !cfg.PMKerneltasksEnabled && isKernelTask(p) {
			continue
		}

		result = append(result, p)
		count++
	}
	return result
}

func toJSON(procs []*ProcStat) string {
	b, err := json.Marshal(procs)
	if err != nil {
		return "{}"
	}

	return string(b)
}
