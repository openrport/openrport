package monitoring

import (
	"context"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api/monitoring"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type Service interface {
	SaveMeasurement(ctx context.Context, measurement *models.Measurement) error
	DeleteMeasurementsOlderThanDays(ctx context.Context, days int64) (int64, error)
	GetClientLatest(ctx context.Context, clientID string) (*models.Measurement, error)
	GetClientMetricsOne(ctx context.Context, clientID string, o *query.Options) (*monitoring.ClientMetricsPayload, error)
	GetClientMetricsList(ctx context.Context, clientID string, o *query.Options) ([]monitoring.ClientMetricsPayload, error)
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

func (s *monitoringService) DeleteMeasurementsOlderThanDays(ctx context.Context, days int64) (int64, error) {
	compare := time.Now().Unix() - (days * 24 * 3600)
	return s.DBProvider.DeleteMeasurementsOlderThan(ctx, compare)
}

func (s *monitoringService) GetClientLatest(ctx context.Context, clientID string) (*models.Measurement, error) {
	return s.DBProvider.GetClientLatest(ctx, clientID)
}

func (s *monitoringService) GetClientMetricsOne(ctx context.Context, clientID string, o *query.Options) (*monitoring.ClientMetricsPayload, error) {
	return s.DBProvider.GetByClientID(ctx, clientID, o)
}

func (s monitoringService) GetClientMetricsList(ctx context.Context, clientID string, o *query.Options) ([]monitoring.ClientMetricsPayload, error) {
	return s.DBProvider.GetListByClientID(ctx, clientID, o)
}
