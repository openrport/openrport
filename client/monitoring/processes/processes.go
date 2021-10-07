package processes

import (
	"runtime"
	"sort"

	"github.com/shirou/gopsutil/mem"

	"github.com/cloudradar-monitoring/rport/client/common"
	"github.com/cloudradar-monitoring/rport/client/monitoring/docker"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type Config struct {
	Enabled                     bool
	EnableKernelTaskMonitoring  bool
	MaxNumberMonitoredProcesses uint
}

type ProcessHandler struct {
	config        Config
	logger        *chshare.Logger
	dockerHandler *docker.Handler
}

func NewProcessHandler(config Config, logger *chshare.Logger, dockerHandler *docker.Handler) *ProcessHandler {
	return &ProcessHandler{config: config, logger: logger, dockerHandler: dockerHandler}
}

func GetDefaultConfig() Config {
	return Config{
		Enabled:                     true,
		EnableKernelTaskMonitoring:  true,
		MaxNumberMonitoredProcesses: 500,
	}
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

// Gets possible process states based on the OS
func getPossibleProcStates() []string {
	fields := []string{
		"blocked",
		"zombie",
		"stopped",
		"running",
		"sleeping",
	}

	switch runtime.GOOS {
	case "windows":
		fields = []string{"running"}
	case "freebsd":
		fields = append(fields, "idle", "wait")
	case "darwin":
		fields = append(fields, "idle")
	case "openbsd":
		fields = append(fields, "idle")
	case "linux":
		fields = append(fields, "dead", "paging", "idle")
	}
	return fields
}

func (ph *ProcessHandler) GetMeasurements(memStat *mem.VirtualMemoryStat) (common.MeasurementsMap, error) {
	var systemMemorySize uint64
	if memStat == nil {
		ph.logger.Debugf("System memory information is unavailable. Some process stats will not be calculated...")
	} else {
		systemMemorySize = memStat.Total
	}
	procs, err := ph.processes(systemMemorySize)
	if err != nil {
		ph.logger.Errorf(err.Error())
		return nil, err
	}

	var m common.MeasurementsMap
	if ph.config.Enabled {
		m = common.MeasurementsMap{"processes": filterProcs(procs, &ph.config)}
	}

	return m, nil
}

func filterProcs(procs []*ProcStat, cfg *Config) []*ProcStat {
	// sort by PID descending:
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].PID > procs[j].PID
	})

	result := make([]*ProcStat, 0, cfg.MaxNumberMonitoredProcesses)
	var count uint
	for _, p := range procs {
		if count == cfg.MaxNumberMonitoredProcesses {
			break
		}

		if !cfg.EnableKernelTaskMonitoring && isKernelTask(p) {
			continue
		}

		result = append(result, p)
		count++
	}
	return result
}
