package monitoring

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/client/system"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type Monitor struct {
	mtx         sync.RWMutex
	conn        ssh.Conn
	logger      *chshare.Logger
	enabled     bool
	interval    time.Duration
	measurement *models.Measurement
	systemInfo  system.SysInfo
}

func NewMonitor(logger *chshare.Logger, enabled bool, interval time.Duration, systemInfo system.SysInfo) *Monitor {
	return &Monitor{logger: logger, enabled: enabled, interval: interval, systemInfo: systemInfo}
}

func (m *Monitor) Start(ctx context.Context) {
	if !m.enabled {
		return
	}

	go m.refreshLoop(ctx)
}

func (m *Monitor) refreshLoop(ctx context.Context) {
	for {
		m.refreshMeasurement(ctx)

		select {
		case <-ctx.Done():
			return
		case <-time.After(m.interval):
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

	newMeasurement.Timestamp = time.Now().Unix()

	cpuPercent, err := m.systemInfo.CPUPercent(ctx)
	if err == nil {
		newMeasurement.CPUUsagePercent = cpuPercent
	}
	memStats, err := m.systemInfo.MemoryStats(ctx)
	if err == nil {
		newMeasurement.MemoryUsagePercent = memStats.UsedPercent
	}
	cpuPercentIOWait, err := m.systemInfo.CPUPercentIOWait(ctx)
	if err == nil {
		newMeasurement.IoUsagePercent = cpuPercentIOWait
	}
	newMeasurement.Processes = `{}`
	newMeasurement.Mountpoints = `{}`
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
