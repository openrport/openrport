package monitoring

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type DBProviderMock struct {
	GraphMetricsListPayload      []*ClientGraphMetricsPayload
	GraphMetricsGraphListPayload []*ClientGraphMetricsGraphPayload
	MetricsListPayload           []*ClientMetricsPayload
	ProcessesListPayload         []*ClientProcessesPayload
	MountpointsListPayload       []*ClientMountpointsPayload
}

func (p *DBProviderMock) CountByClientID(ctx context.Context, clientID string, fo *query.ListOptions) (int, error) {
	return 10, nil
}

func (p *DBProviderMock) ListProcessesByClientID(ctx context.Context, clientID string, fo *query.ListOptions) ([]*ClientProcessesPayload, error) {
	return p.ProcessesListPayload, nil
}

func (p *DBProviderMock) ListMountpointsByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]*ClientMountpointsPayload, error) {
	return p.MountpointsListPayload, nil
}

func (p *DBProviderMock) ListGraphByClientID(context.Context, string, float64, *query.ListOptions, string) ([]*ClientGraphMetricsGraphPayload, error) {
	return p.GraphMetricsGraphListPayload, nil
}

func (p *DBProviderMock) ListMetricsByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]*ClientMetricsPayload, error) {
	return p.MetricsListPayload, nil
}

func (p *DBProviderMock) ListGraphMetricsByClientID(ctx context.Context, clientID string, hours float64, o *query.ListOptions) ([]*ClientGraphMetricsPayload, error) {
	return p.GraphMetricsListPayload, nil
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
