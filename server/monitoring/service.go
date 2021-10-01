package monitoring

import (
	"context"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type Service interface {
	SaveMeasurement(measurement *models.Measurement) error
}

type monitoringService struct {
	DBProvider DBProvider
}

func NewService(dbProvider DBProvider) Service {
	return &monitoringService{DBProvider: dbProvider}
}
func (s *monitoringService) SaveMeasurement(measurement *models.Measurement) error {
	return s.DBProvider.CreateMeasurement(context.Background(), measurement)
}
