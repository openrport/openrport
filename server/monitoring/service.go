package monitoring

import (
	"context"
	"github.com/cloudradar-monitoring/rport/share/models"
	"time"
)

type Service interface {
	SaveMeasurement(ctx context.Context, measurement *models.Measurement) error
	Cleanup(ctx context.Context, days int64) error
}

type monitoringService struct {
	DBProvider DBProvider
}

func NewService(dbProvider DBProvider) Service {
	return &monitoringService{DBProvider: dbProvider}
}
func (s *monitoringService) SaveMeasurement(ctx context.Context, measurement *models.Measurement) error {
	return s.DBProvider.CreateMeasurement(ctx, measurement)
}

func (s *monitoringService) Cleanup(ctx context.Context, days int64) error {
	compare := time.Now().Unix() - (days * 3600)
	return s.DBProvider.DeleteMeasurementsOlderThan(ctx, compare)
}
