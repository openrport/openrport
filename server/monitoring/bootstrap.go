package monitoring

import (
	"github.com/realvnc-labs/rport/share/logger"
)

const (
	QueueLoggerName = "measurements-queue"
	QueueSize       = 10000
)

type monitoringWithQueuing struct {
	Service
	MeasurementSaver
}

type Monitoring interface {
	Service
	MeasurementSaver
}

func BootstrapMonitoring(logger *logger.Logger, dbProvider DBProvider) Monitoring {
	service := NewService(dbProvider)
	return &monitoringWithQueuing{
		Service:          service,
		MeasurementSaver: NewMeasurementQueuing(logger.Fork(QueueLoggerName), service, QueueSize),
	}
}
