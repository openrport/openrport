package monitoring

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

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
}

func NewMonitor(logger *chshare.Logger, enabled bool, interval time.Duration) *Monitor {
	return &Monitor{logger: logger, enabled: enabled, interval: interval}
}

func (m *Monitor) Start(ctx context.Context) {
	if !m.enabled {
		return
	}

	go m.refreshLoop(ctx)
}

func (m *Monitor) refreshLoop(ctx context.Context) {
	for {
		m.refreshMeasurement()

		select {
		case <-ctx.Done():
			return
		case <-time.After(m.interval):
		}
	}
}

func (m *Monitor) refreshMeasurement() {
	m.mtx.Lock()
	m.measurement = createMeasurement()
	m.mtx.Unlock()

	go m.sendMeasurement()
}

func createMeasurement() *models.Measurement {
	var newMeasurement = &models.Measurement{}

	newMeasurement.Timestamp = time.Now().Unix()
	newMeasurement.CPUUsagePercent = 10.0
	newMeasurement.MemoryUsagePercent = 50.0
	newMeasurement.IoUsagePercent = 30.0
	newMeasurement.Processes = `{}`
	newMeasurement.Mountpoints = `{}`
	return newMeasurement
}

// sendMeasurement sends system measurement data to server
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
