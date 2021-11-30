package monitoring

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/client/monitoring/config"
	"github.com/cloudradar-monitoring/rport/client/monitoring/docker"
	"github.com/cloudradar-monitoring/rport/client/monitoring/fs"
	"github.com/cloudradar-monitoring/rport/client/monitoring/processes"
	"github.com/cloudradar-monitoring/rport/client/system"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type Monitor struct {
	mtx               sync.RWMutex
	conn              ssh.Conn
	stopFn            func()
	logger            *chshare.Logger
	config            config.MonitoringConfig
	measurement       *models.Measurement
	systemInfo        system.SysInfo
	fileSystemWatcher *fs.FileSystemWatcher
	processHandler    *processes.ProcessHandler
}

func NewMonitor(logger *chshare.Logger, config config.MonitoringConfig, systemInfo system.SysInfo) *Monitor {
	fsWatcher := fs.NewWatcher(fs.FileSystemWatcherConfig{
		TypeInclude:                 config.FSTypeInclude,
		PathExclude:                 config.FSPathExclude,
		PathExcludeRecurse:          config.FSPathExcludeRecurse,
		Metrics:                     fs.DefaultMetrics(),
		IdentifyMountpointsByDevice: config.FSIdentifyMountpointsByDevice,
	}, logger)
	dockerHandler := docker.NewHandler(logger)
	processHandler := processes.NewProcessHandler(config, logger, dockerHandler)
	return &Monitor{logger: logger, config: config, systemInfo: systemInfo, fileSystemWatcher: fsWatcher, processHandler: processHandler}
}

func (m *Monitor) Start(ctx context.Context) {
	if !m.config.Enabled {
		return
	}

	ctx, m.stopFn = context.WithCancel(ctx)

	go m.refreshLoop(ctx)
	m.logger.Debugf("Monitor started")
}

func (m *Monitor) Stop() {
	if m.stopFn == nil {
		return
	}

	m.stopFn()
	m.logger.Debugf("Monitor stopped")
}

func (m *Monitor) refreshLoop(ctx context.Context) {
	for {
		m.refreshMeasurement(ctx)

		select {
		case <-ctx.Done():
			m.logger.Errorf("Monitoring ended by context.Done")
			return
		case <-time.After(m.config.Interval):
		}
	}
}

func (m *Monitor) refreshMeasurement(ctx context.Context) {
	m.mtx.Lock()
	m.measurement = m.createMeasurement(ctx)
	m.mtx.Unlock()

	go m.sendMeasurement()
}

func (m *Monitor) createMeasurement(ctx context.Context) *models.Measurement {
	var newMeasurement = &models.Measurement{}

	newMeasurement.Timestamp = time.Now().UTC()

	cpuPercent, err := m.systemInfo.CPUPercent(ctx)
	if err == nil {
		newMeasurement.CPUUsagePercent = cpuPercent
	} else {
		m.logger.Debugf("Cannot measure cpu_usage_percent:" + err.Error())
	}
	memStats, err := m.systemInfo.MemoryStats(ctx)
	if err == nil {
		newMeasurement.MemoryUsagePercent = memStats.UsedPercent
	} else {
		m.logger.Debugf("Cannot measure memory_usage_percent:" + err.Error())
	}
	cpuPercentIOWait, err := m.systemInfo.CPUPercentIOWait(ctx)
	if err == nil {
		newMeasurement.IoUsagePercent = cpuPercentIOWait
	} else {
		m.logger.Debugf("Cannot measure io_usage_percent:" + err.Error())
	}

	processes, err := m.processHandler.GetProcessesJSON(memStats)
	if err == nil {
		newMeasurement.Processes = processes
	} else {
		m.logger.Debugf("Cannot measure processes:" + err.Error())
	}

	fsMap, err := m.fileSystemWatcher.Results()
	if err == nil {
		newMeasurement.Mountpoints = fsMap.ToJSON()
	} else {
		m.logger.Debugf("Cannot measure mountpoints:" + err.Error())
	}

	return newMeasurement
}

// sends system measurement data to server using ssh-connection
func (m *Monitor) sendMeasurement() {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	if m.conn != nil && m.measurement != nil {
		data, err := json.Marshal(m.measurement)
		if err != nil {
			m.logger.Errorf("Could not marshal json for save_measurement: %v", err)
			return
		}

		m.logger.Debugf("sending %d bytes of monitoring data to server", len(data))
		_, _, err = m.conn.SendRequest(comm.RequestTypeSaveMeasurement, false, data)
		if err != nil {
			m.logger.Errorf("Could not send save_measurement: %v", err)
			return
		}
	}
}

func (m *Monitor) SetConn(c ssh.Conn) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.conn = c
}
