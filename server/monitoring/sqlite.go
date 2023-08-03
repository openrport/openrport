package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/realvnc-labs/rport/db/migration/monitoring"
	"github.com/realvnc-labs/rport/db/sqlite"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
	"github.com/realvnc-labs/rport/share/query"
)

type DBProvider interface {
	CreateMeasurement(ctx context.Context, measurement *models.Measurement) error
	DeleteMeasurementsBefore(ctx context.Context, compare time.Time) (int64, error)
	ListGraphByClientID(context.Context, string, float64, *query.ListOptions, string) ([]*ClientGraphMetricsGraphPayload, error)
	ListGraphMetricsByClientID(context.Context, string, float64, *query.ListOptions) ([]*ClientGraphMetricsPayload, error)
	ListMetricsByClientID(context.Context, string, *query.ListOptions) ([]*ClientMetricsPayload, error)
	ListMountpointsByClientID(context.Context, string, *query.ListOptions) ([]*ClientMountpointsPayload, error)
	ListProcessesByClientID(context.Context, string, *query.ListOptions) ([]*ClientProcessesPayload, error)
	CountByClientID(context.Context, string, *query.ListOptions) (int, error)
	Close() error
}

// MaxDeletedEntries to prevent "stop the world" after longer restart, when there is a lot of measurements to clean up
// clean them in chunks and this is the chunk size
const MaxDeletedEntries = 5000

type SqliteProvider struct {
	db        *sqlx.DB
	logger    *logger.Logger
	converter *query.SQLConverter
}

func NewSqliteProvider(dbPath string, dataSourceOptions sqlite.DataSourceOptions, logger *logger.Logger) (DBProvider, error) {
	db, err := sqlite.New(dbPath, monitoring.AssetNames(), monitoring.Asset, dataSourceOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring DB instance: %v", err)
	}

	logger.Infof("initialized database at %s", dbPath)

	return &SqliteProvider{
		db:        db,
		logger:    logger,
		converter: query.NewSQLConverter(db.DriverName()),
	}, nil
}

func (p *SqliteProvider) ListMountpointsByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]*ClientMountpointsPayload, error) {
	q := "SELECT * FROM `measurements` as `mountpoints` WHERE `client_id` = ? "
	params := []interface{}{}
	params = append(params, clientID)
	q, params = p.converter.AppendOptionsToQuery(o, q, params)

	val := []*ClientMountpointsPayload{}
	err := p.db.SelectContext(ctx, &val, q, params...)
	return val, err
}

func (p *SqliteProvider) ListProcessesByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]*ClientProcessesPayload, error) {
	q := "SELECT * FROM `measurements` as `processes` WHERE `client_id` = ? "
	params := []interface{}{}
	params = append(params, clientID)
	q, params = p.converter.AppendOptionsToQuery(o, q, params)

	val := []*ClientProcessesPayload{}
	err := p.db.SelectContext(ctx, &val, q, params...)
	return val, err
}

func (p *SqliteProvider) ListMetricsByClientID(ctx context.Context, clientID string, o *query.ListOptions) ([]*ClientMetricsPayload, error) {
	q := "SELECT * FROM `measurements` as `metrics` WHERE `client_id` = ? "
	params := []interface{}{}
	params = append(params, clientID)
	q, params = p.converter.AppendOptionsToQuery(o, q, params)

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
	q, params = p.converter.AppendOptionsToQuery(&countOptions, q, params)

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

	q, params = p.converter.AddWhere(lo.Filters, q, params)

	/*This is the part of "downsampling graph data" (group together graph points, so that you don't get too much points in one request).
	The value of "29" comes from Thorsten. He did some research and found out that "29" would be the best fit.
	*/
	q = q + ` GROUP BY round((strftime('%s',timestamp)/(?)),0)`
	divisor := (math.Round(hours*100) / 100) * 29
	params = append(params, divisor)

	q = p.converter.AddOrderBy(lo.Sorts, q)

	val := []*ClientGraphMetricsPayload{}
	err := p.db.SelectContext(ctx, &val, q, params...)
	return val, err
}

func (p *SqliteProvider) ListGraphByClientID(ctx context.Context, clientID string, hours float64, lo *query.ListOptions, graph string) ([]*ClientGraphMetricsGraphPayload, error) {
	params := []interface{}{}
	params = append(params, clientID)
	field, okField := ClientGraphNameToField[graph]
	alias, okAlias := ClientGraphNameToAlias[graph]
	if !okField || !okAlias {
		return nil, fmt.Errorf("unknown graph: %s", graph)
	}

	q := `SELECT timestamp, `
	q = q + ` 
		round(avg(` + field + `),2) as ` + alias + `_avg,
		min(` + field + `) as ` + alias + `_min,
		max(` + field + `) as ` + alias + `_max`

	if strings.HasPrefix(graph, "net_") {
		field = strings.ReplaceAll(field, "_in", "_out")
		alias = strings.ReplaceAll(alias, "_in", "_out")
		q = q + `, 
		round(avg(` + field + `),2) as ` + alias + `_avg,
		min(` + field + `) as ` + alias + `_min,
		max(` + field + `) as ` + alias + `_max`
	}
	q = q + ` 
	FROM measurements WHERE client_id = ?`

	q, params = p.converter.AddWhere(lo.Filters, q, params)

	q = q + ` GROUP BY round((strftime('%s',timestamp)/(?)),0)`
	divisor := (math.Round(hours*100) / 100) * 29
	params = append(params, divisor)

	query := p.converter.AddOrderBy(lo.Sorts, q)

	val := []*ClientGraphMetricsGraphPayload{}
	err := p.db.SelectContext(ctx, &val, query, params...)
	return val, err
}

func (p *SqliteProvider) CreateMeasurement(ctx context.Context, measurement *models.Measurement) error {
	q := `INSERT INTO measurements (client_id, timestamp, cpu_usage_percent, memory_usage_percent, io_usage_percent, processes, mountpoints, net_lan_in, net_lan_out, net_wan_in, net_wan_out) 
		VALUES (:client_id, :timestamp, :cpu_usage_percent, :memory_usage_percent, :io_usage_percent, :processes, :mountpoints, `
	if measurement.NetLan == nil {
		q = q + `null, null, `
	} else {
		q = q + `:net_lan.in, :net_lan.out, `
	}
	if measurement.NetWan == nil {
		q = q + `null, null`
	} else {
		q = q + `:net_wan.in, :net_wan.out`
	}
	query := q + ")"

	_, err := sqlite.WithRetryWhenBusy(func() (result sql.Result, err error) {
		result, err = p.db.NamedExecContext(ctx, query, measurement)
		return result, err
	}, "createmeasurement", p.logger)

	return err
}

// DeleteMeasurementsBefore deletes entries in chunks of MaxDeletedEntries
// to clean all you can run in loop as long as there are more than 0 rows affected
func (p *SqliteProvider) DeleteMeasurementsBefore(ctx context.Context, compare time.Time) (int64, error) {
	result, err := p.db.ExecContext(ctx, "DELETE FROM measurements WHERE  timestamp IN (SELECT distinct timestamp FROM measurements WHERE timestamp < ? ORDER BY timestamp LIMIT ?)", compare, MaxDeletedEntries)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (p *SqliteProvider) Close() error {
	return p.db.Close()
}
