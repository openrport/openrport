package monitoring

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	monitoring_api "github.com/cloudradar-monitoring/rport/server/api/monitoring"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type DBProviderMock struct {
	MetricsPayload     *monitoring_api.ClientMetricsPayload
	MetricsListPayload []monitoring_api.ClientMetricsPayload
	ProcessesPayload   *monitoring_api.ClientProcessesPayload
	MountpointsPayload *monitoring_api.ClientMountpointsPayload
}

func (p *DBProviderMock) GetProcessesLatestByClientID(ctx context.Context, clientID string) (*monitoring_api.ClientProcessesPayload, error) {
	return p.ProcessesPayload, nil
}

func (p *DBProviderMock) GetProcessesNearestByClientID(ctx context.Context, clientID string, timestamp time.Time) (*monitoring_api.ClientProcessesPayload, error) {
	return p.ProcessesPayload, nil
}

func (p *DBProviderMock) GetMountpointsLatestByClientID(ctx context.Context, clientID string) (*monitoring_api.ClientMountpointsPayload, error) {
	return p.MountpointsPayload, nil
}

func (p *DBProviderMock) GetMountpointsNearestByClientID(ctx context.Context, clientID string, timestamp time.Time) (*monitoring_api.ClientMountpointsPayload, error) {
	return p.MountpointsPayload, nil
}

func (p *DBProviderMock) GetMetricsLatestByClientID(ctx context.Context, clientID string, fields []query.FieldsOption) (val *monitoring_api.ClientMetricsPayload, err error) {
	return p.MetricsPayload, nil
}

func (p *DBProviderMock) GetMetricsListByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring_api.ClientMetricsPayload, error) {
	return p.MetricsListPayload, nil
}

func (p *DBProviderMock) GetMetricsListDownsampledByClientID(ctx context.Context, clientID string, hours float64, o *query.ListOptions) ([]monitoring_api.ClientMetricsPayload, error) {
	return p.MetricsListPayload, nil
}

func (p *DBProviderMock) CreateMeasurement(ctx context.Context, measurement *models.Measurement) error {
	return nil
}

func (p *DBProviderMock) DeleteMeasurementsBefore(ctx context.Context, compare time.Time) (int64, error) {
	return 0, nil
}

func (p *DBProviderMock) Close() error {
	return nil
}

func (p *DBProviderMock) DB() *sqlx.DB {
	return nil
}
