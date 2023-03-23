package monitoring

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/realvnc-labs/rport/client/monitoring/fs"
	"github.com/realvnc-labs/rport/client/monitoring/networking"
	"github.com/realvnc-labs/rport/client/monitoring/processes"
	"github.com/realvnc-labs/rport/client/system"
	"github.com/realvnc-labs/rport/share/clientconfig"
	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
)

type Monitor struct {
	mtx               sync.RWMutex
	conn              ssh.Conn
	stopFn            func()
	logger            *logger.Logger
	config            clientconfig.MonitoringConfig
	measurement       *models.Measurement
	systemInfo        system.SysInfo
	fileSystemWatcher *fs.FileSystemWatcher
	processHandler    *processes.ProcessHandler
	netHandler        *networking.NetHandler
}

func NewMonitor(logger *logger.Logger, config clientconfig.MonitoringConfig, systemInfo system.SysInfo) *Monitor {
	fsWatcher := fs.NewWatcher(fs.FileSystemWatcherConfig{
		TypeInclude:                 config.FSTypeInclude,
		PathExclude:                 config.FSPathExclude,
		PathExcludeRecurse:          config.FSPathExcludeRecurse,
		Metrics:                     fs.DefaultMetrics(),
		IdentifyMountpointsByDevice: config.FSIdentifyMountpointsByDevice,
	}, logger)
	processHandler := processes.NewProcessHandler(config, logger)
	netHandler := networking.NewNetHandler(&config)
	return &Monitor{logger: logger, config: config, systemInfo: systemInfo, fileSystemWatcher: fsWatcher, processHandler: processHandler, netHandler: netHandler}
}

func (m *Monitor) Start(ctx context.Context) {
	if !m.config.Enabled {
		return
	}

	ctx, m.stopFn = context.WithCancel(ctx)

	go m.refreshLoop(ctx)
	m.logger.Debugf("Monitoring started")
}

func (m *Monitor) Stop() {
	if m.stopFn == nil {
		return
	}

	m.stopFn()
	m.logger.Debugf("Monitoring stopped")
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

	netLan, netWan, err := m.netHandler.GetNets()
	if err == nil {
		newMeasurement.NetLan = netLan
		newMeasurement.NetWan = netWan
	} else {
		m.logger.Debugf("Cannot measure network bandwidth:" + err.Error())
	}
	return newMeasurement
}

// sends system measurement data to server using ssh-connection
func (m *Monitor) sendMeasurement() {
	t0 := time.Now()
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	if m.conn == nil {
		m.logger.Debugf("Cannot send measurement. SSH connection missing. m.conn = nil")
	}

	if m.conn != nil && m.measurement != nil {
		data, err := json.Marshal(m.measurement)
		if err != nil {
			m.logger.Errorf("Could not marshal json for save_measurement: %v", err)
			return
		}

		_, _, err = m.conn.SendRequest(comm.RequestTypeSaveMeasurement, false, data)
		if err != nil {
			m.logger.Errorf("Could not send save_measurement: %v", err)
			return
		}
		m.logger.Debugf("%d bytes of monitoring measurements sent within %s", len(data), time.Since(t0))
	}

}

func (m *Monitor) SetConn(c ssh.Conn) {
	m.logger.Debugf("SSH Connection for monitoring set.")
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.conn = c
}
