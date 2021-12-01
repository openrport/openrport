package monitoring

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"

	monitoring "github.com/cloudradar-monitoring/rport/db/migration/monitoring"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type DBProvider interface {
	CreateMeasurement(ctx context.Context, measurement *models.Measurement) error
	DeleteMeasurementsBefore(ctx context.Context, compare time.Time) (int64, error)
	ListGraphMetricsByClientID(context.Context, string, float64, *query.ListOptions) ([]*ClientGraphMetricsPayload, error)
	ListMetricsByClientID(context.Context, string, *query.ListOptions) ([]*ClientMetricsPayload, error)
	ListMountpointsByClientID(context.Context, string, *query.ListOptions) ([]*ClientMountpointsPayload, error)
	ListProcessesByClientID(context.Context, string, *query.ListOptions) ([]*ClientProcessesPayload, error)
	CountByClientID(context.Context, string, *query.ListOptions) (int, error)
	Close() error
}

type SqliteProvider struct {
	db     *sqlx.DB
	logger *logger.Logger
}

func NewSqliteProvider(dbPath string, logger *logger.Logger) (DBProvider, error) {
	db, err := sqlite.New(dbPath, monitoring.AssetNames(), monitoring.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring DB instance: %v", err)
	}

	logger.Infof("initialized database at %s", dbPath)

	return &SqliteProvider{db: db, logger: logger}, nil
}

func (p *SqliteProvider) ListMountpointsByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]*ClientMountpointsPayload, error) {
	q := "SELECT * FROM `measurements` as `mountpoints` WHERE `client_id` = ? "
	params := []interface{}{}
	params = append(params, clientID)
	q, params = query.AppendOptionsToQuery(o, q, params)

	val := []*ClientMountpointsPayload{}
	err := p.db.SelectContext(ctx, &val, q, params...)
	return val, err
}

func (p *SqliteProvider) ListProcessesByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]*ClientProcessesPayload, error) {
	q := "SELECT * FROM `measurements` as `processes` WHERE `client_id` = ? "
	params := []interface{}{}
	params = append(params, clientID)
	q, params = query.AppendOptionsToQuery(o, q, params)

	val := []*ClientProcessesPayload{}
	err := p.db.SelectContext(ctx, &val, q, params...)
	return val, err
}

func (p *SqliteProvider) ListMetricsByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]*ClientMetricsPayload, error) {
	q := "SELECT * FROM `measurements` as `metrics` WHERE `client_id` = ? "
	params := []interface{}{}
	params = append(params, clientID)
	q, params = query.AppendOptionsToQuery(o, q, params)

	val := []*ClientMetricsPayload{}
	err := p.db.SelectContext(ctx, &val, q, params...)
	return val, err
}

func (p *SqliteProvider) CountByClientID(ctx context.Context, clientID string, options *query.ListOptions) (int, error) {
	var result int

	q := "SELECT COUNT(*) FROM `measurements` WHERE `client_id` = ? "
	countOptions := *options
	countOptions.Pagination = nil

	params := []interface{}{}
	params = append(params, clientID)
	q, params = query.AppendOptionsToQuery(&countOptions, q, params)

	err := p.db.GetContext(ctx, &result, q, params...)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (p *SqliteProvider) ListGraphMetricsByClientID(ctx context.Context, clientID string, hours float64, lo *query.ListOptions) ([]*ClientGraphMetricsPayload, error) {
	params := []interface{}{}
	params = append(params, clientID)

	q := `SELECT
		timestamp,
		round(avg(cpu_usage_percent),2) as cpu_usage_percent_avg,
		min(cpu_usage_percent) as cpu_usage_percent_min,
		max(cpu_usage_percent) as cpu_usage_percent_max,
		round(avg(memory_usage_percent),2) as memory_usage_percent_avg,
		min(memory_usage_percent) as memory_usage_percent_min,
		max(memory_usage_percent) as memory_usage_percent_max,
		round(avg(io_usage_percent),2) as io_usage_percent_avg,
		min(io_usage_percent) as io_usage_percent_min,
		max(io_usage_percent) as io_usage_percent_max
	FROM measurements WHERE client_id = ?`

	q, params = query.AddWhere(lo.Filters, q, params)

	q = q + ` GROUP BY round((strftime('%s',timestamp)/(?)),0)`
	divisor := (math.Round(hours*100) / 100) * 29
	params = append(params, divisor)

	q = query.AddOrderBy(lo.Sorts, q)

	val := []*ClientGraphMetricsPayload{}
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
