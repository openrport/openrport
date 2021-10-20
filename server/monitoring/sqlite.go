package monitoring

import (
	"context"
	"fmt"

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
	DeleteMeasurementsOlderThan(ctx context.Context, days int64) (int64, error)
	GetClientLatest(ctx context.Context, clientID string) (*models.Measurement, error)
	GetMetricsByClientID(ctx context.Context, clientID string, o *query.Options) (val *monitoring2.ClientMetricsPayload, err error)
	GetMetricsListByClientID(ctx context.Context, clientID string, o *query.Options) (val []monitoring2.ClientMetricsPayload, err error)
	GetProcessesLatestByClientID(ctx context.Context, clientID string) (val *monitoring2.ClientProcessesPayload, err error)
	GetProcessesNearestByClientID(ctx context.Context, clientID string, at string) (val *monitoring2.ClientProcessesPayload, err error)
	GetMountpointsLatestByClientID(ctx context.Context, clientID string) (val *monitoring2.ClientMountpointsPayload, err error)
	GetMountpointsNearestByClientID(ctx context.Context, clientID string, at string) (val *monitoring2.ClientMountpointsPayload, err error)
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
	err := p.db.Get(&payload, "SELECT strftime('%Y-%m-%d %H:%M:%S',datetime(timestamp, 'unixepoch')) as date, processes FROM measurements WHERE client_id = ? ORDER BY timestamp DESC LIMIT 1", clientID)
	return &payload, err
}

func (p *SqliteProvider) GetProcessesNearestByClientID(ctx context.Context, clientID string, timestamp string) (*monitoring2.ClientProcessesPayload, error) {
	var payload monitoring2.ClientProcessesPayload
	err := p.db.Get(&payload, "SELECT strftime('%Y-%m-%d %H:%M:%S',datetime(timestamp, 'unixepoch')) as date, processes FROM measurements WHERE client_id = ?  AND timestamp >= ? LIMIT 1", clientID, timestamp)
	return &payload, err
}

func (p *SqliteProvider) GetMountpointsLatestByClientID(ctx context.Context, clientID string) (*monitoring2.ClientMountpointsPayload, error) {
	var payload monitoring2.ClientMountpointsPayload
	err := p.db.Get(&payload, "SELECT strftime('%Y-%m-%d %H:%M:%S',datetime(timestamp, 'unixepoch')) as date, mountpoints FROM measurements WHERE client_id = ? ORDER BY timestamp DESC LIMIT 1", clientID)
	return &payload, err
}

func (p *SqliteProvider) GetMountpointsNearestByClientID(ctx context.Context, clientID string, timestamp string) (*monitoring2.ClientMountpointsPayload, error) {
	var payload monitoring2.ClientMountpointsPayload
	err := p.db.Get(&payload, "SELECT strftime('%Y-%m-%d %H:%M:%S',datetime(timestamp, 'unixepoch')) as date, mountpoints FROM measurements WHERE client_id = ?  AND timestamp >= ? LIMIT 1", clientID, timestamp)
	return &payload, err
}

func (p *SqliteProvider) GetMetricsByClientID(ctx context.Context, clientID string, o *query.Options) (val *monitoring2.ClientMetricsPayload, err error) {
	q := "SELECT * FROM `measurements` as `metrics` WHERE `client_id` = ? "
	params := []interface{}{}
	q, _ = query.ConvertOptionsToQuery(o, q, params)
	q = q + " LIMIT 1"

	val = new(monitoring2.ClientMetricsPayload)
	err = p.db.GetContext(ctx, val, q, clientID)
	return val, err
}

func (p *SqliteProvider) GetMetricsListByClientID(ctx context.Context, clientID string, o *query.Options) ([]monitoring2.ClientMetricsPayload, error) {
	q := "SELECT * FROM `measurements` as `metrics` WHERE `client_id` = ? "
	params := []interface{}{}
	params = append(params, clientID)
	q, params = query.ConvertOptionsToQuery(o, q, params)

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

func (p *SqliteProvider) DeleteMeasurementsOlderThan(ctx context.Context, compare int64) (int64, error) {
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
