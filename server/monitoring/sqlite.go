package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	monitoring "github.com/cloudradar-monitoring/rport/db/migration/monitoring"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	monitoring2 "github.com/cloudradar-monitoring/rport/server/api/monitoring"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type DBProvider interface {
	CreateMeasurement(ctx context.Context, measurement *models.Measurement) error
	DeleteMeasurementsBefore(ctx context.Context, compare time.Time) (int64, error)
	GetClientLatest(ctx context.Context, clientID string) (*models.Measurement, error)
	GetMetricsLatestByClientID(ctx context.Context, clientID string, fields []query.FieldsOption) (*monitoring2.ClientMetricsPayload, error)
	GetMetricsListByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring2.ClientMetricsPayload, error)
	GetMetricsSinceLimitedByClientID(ctx context.Context, clientID string, filters []query.FilterOption) ([]monitoring2.ClientMetricsPayload, error)
	GetProcessesLatestByClientID(ctx context.Context, clientID string) (*monitoring2.ClientProcessesPayload, error)
	GetProcessesNearestByClientID(ctx context.Context, clientID string, at time.Time) (*monitoring2.ClientProcessesPayload, error)
	GetMountpointsLatestByClientID(ctx context.Context, clientID string) (*monitoring2.ClientMountpointsPayload, error)
	GetMountpointsNearestByClientID(ctx context.Context, clientID string, at time.Time) (*monitoring2.ClientMountpointsPayload, error)
	Close() error
	DB() *sqlx.DB
}

type SqliteProvider struct {
	db     *sqlx.DB
	logger *chshare.Logger
}

func NewSqliteProvider(dbPath string, logger *chshare.Logger) (DBProvider, error) {
	db, err := sqlite.New(dbPath, monitoring.AssetNames(), monitoring.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring DB instance: %v", err)
	}

	logger.Infof("initialized database at %s", dbPath)

	return &SqliteProvider{db: db, logger: logger}, nil
}

func (p *SqliteProvider) GetProcessesLatestByClientID(ctx context.Context, clientID string) (*monitoring2.ClientProcessesPayload, error) {
	var payload monitoring2.ClientProcessesPayload
	err := p.db.Get(&payload, "SELECT timestamp, processes FROM measurements WHERE client_id = ? ORDER BY timestamp DESC LIMIT 1", clientID)
	return &payload, err
}

func (p *SqliteProvider) GetProcessesNearestByClientID(ctx context.Context, clientID string, timestamp time.Time) (*monitoring2.ClientProcessesPayload, error) {
	var payload monitoring2.ClientProcessesPayload
	err := p.db.Get(&payload, "SELECT timestamp, processes FROM measurements WHERE client_id = ?  AND timestamp >= ? LIMIT 1", clientID, timestamp)
	return &payload, err
}

func (p *SqliteProvider) GetMountpointsLatestByClientID(ctx context.Context, clientID string) (*monitoring2.ClientMountpointsPayload, error) {
	var payload monitoring2.ClientMountpointsPayload
	err := p.db.Get(&payload, "SELECT timestamp, mountpoints FROM measurements WHERE client_id = ? ORDER BY timestamp DESC LIMIT 1", clientID)
	return &payload, err
}

func (p *SqliteProvider) GetMountpointsNearestByClientID(ctx context.Context, clientID string, timestamp time.Time) (*monitoring2.ClientMountpointsPayload, error) {
	var payload monitoring2.ClientMountpointsPayload
	err := p.db.Get(&payload, "SELECT timestamp, mountpoints FROM measurements WHERE client_id = ?  AND timestamp >= ? LIMIT 1", clientID, timestamp)
	return &payload, err
}

func (p *SqliteProvider) GetMetricsLatestByClientID(ctx context.Context, clientID string, fields []query.FieldsOption) (val *monitoring2.ClientMetricsPayload, err error) {
	q := "SELECT * FROM `measurements` as `metrics` WHERE `client_id` = ? ORDER BY timestamp DESC LIMIT 1"
	q = query.ReplaceStarSelect(fields, q)

	val = new(monitoring2.ClientMetricsPayload)
	err = p.db.GetContext(ctx, val, q, clientID)
	return val, err
}

func (p *SqliteProvider) GetMetricsListByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring2.ClientMetricsPayload, error) {
	q := "SELECT * FROM `measurements` as `metrics` WHERE `client_id` = ? "
	params := []interface{}{}
	params = append(params, clientID)
	q, params = query.AppendOptionsToQuery(o, q, params)

	val := []monitoring2.ClientMetricsPayload{}
	err := p.db.SelectContext(ctx, &val, q, params...)
	return val, err
}

func (p *SqliteProvider) GetMetricsSinceLimitedByClientID(ctx context.Context, clientID string, filters []query.FilterOption) ([]monitoring2.ClientMetricsPayload, error) {
	q := "SELECT * FROM `measurements` as `metrics` WHERE `client_id` = ? "
	params := []interface{}{}
	params = append(params, clientID)
	params = append(params, filters[0].Values[0])

	val := []monitoring2.ClientMetricsPayload{}
	err := p.db.SelectContext(ctx, &val, q, params...)
	return val, err
}

func (p *SqliteProvider) GetClientLatest(ctx context.Context, clientID string) (*models.Measurement, error) {
	var m models.Measurement
	err := p.db.Get(&m, "SELECT * FROM measurements WHERE client_id = ? ORDER BY timestamp DESC LIMIT 1", clientID)
	return &m, err
}

func (p *SqliteProvider) CreateMeasurement(ctx context.Context, measurement *models.Measurement) error {
	_, err := p.db.NamedExecContext(
		ctx,
		"INSERT INTO measurements (client_id, timestamp, cpu_usage_percent, memory_usage_percent, io_usage_percent, processes, mountpoints) "+
			"VALUES (:client_id, :timestamp, :cpu_usage_percent, :memory_usage_percent, :io_usage_percent, :processes, :mountpoints)",
		measurement,
	)
	return err
}

func (p *SqliteProvider) DeleteMeasurementsBefore(ctx context.Context, compare time.Time) (int64, error) {
	result, err := p.db.ExecContext(ctx, "DELETE FROM measurements WHERE  timestamp < ?", compare)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *SqliteProvider) Close() error {
	return p.db.Close()
}

func (p *SqliteProvider) DB() *sqlx.DB {
	return p.db
}
