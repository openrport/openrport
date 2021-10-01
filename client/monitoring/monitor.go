package monitoring

import (
	"encoding/json"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"golang.org/x/crypto/ssh"
	"sync"
)

type Monitor struct {
	mtx    sync.RWMutex
	conn   ssh.Conn
	logger *chshare.Logger
}

// sendMeasurement sends measurement data in background
func (m *Monitor) sendMeasurement(measurement *models.Measurement) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	if m.conn != nil {
		data, err := json.Marshal(measurement)
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
