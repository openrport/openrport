package monitoring

import (
	"context"
	"fmt"
	"math"
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
	GetMetricsLatestByClientID(ctx context.Context, clientID string, fields []query.FieldsOption) (*monitoring2.ClientMetricsPayload, error)
	GetMetricsListByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring2.ClientMetricsPayload, error)
	GetMetricsListDownsampledByClientID(ctx context.Context, clientID string, hours float64, o *query.ListOptions) ([]monitoring2.ClientMetricsPayload, error)
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

func (p *SqliteProvider) GetMetricsListDownsampledByClientID(ctx context.Context, clientID string, hours float64, o *query.ListOptions) ([]monitoring2.ClientMetricsPayload, error) {
	q := `SELECT
		timestamp,
		round(avg(cpu_usage_percent),2) as cpu_usage_percent,
		min(cpu_usage_percent) as cpu_usage_percent_min,
		max(cpu_usage_percent) as cpu_usage_percent_max,
		round(avg(memory_usage_percent),2) as memory_usage_percent,
		min(memory_usage_percent) as memory_usage_percent_min,
		max(memory_usage_percent) as memory_usage_percent_max,
		round(avg(io_usage_percent),2) as io_usage_percent,
		min(io_usage_percent) as io_usage_percent_min,
		max(io_usage_percent) as io_usage_percent_max
	FROM measurements
	WHERE client_id = ? and timestamp >= ? and timestamp <= ?
	GROUP BY round((strftime('%s',timestamp)/(?)),0)
	ORDER BY timestamp DESC`

	params := []interface{}{}
	params = append(params, clientID)
	params = append(params, o.Filters[0].Values[0])
	params = append(params, o.Filters[1].Values[0])
	divisor := math.Round(hours*100) / 100 * 29
	params = append(params, divisor)

	val := []monitoring2.ClientMetricsPayload{}
	err := p.db.SelectContext(ctx, &val, q, params...)
	return val, err
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
