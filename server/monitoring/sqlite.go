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
	GetByClientID(ctx context.Context, clientID string, o *query.Options) (val *monitoring2.ClientMetricsPayload, err error)
	GetListByClientID(ctx context.Context, clientID string, o *query.Options) (val []monitoring2.ClientMetricsPayload, err error)
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

func (p *SqliteProvider) GetByClientID(ctx context.Context, clientID string, o *query.Options) (val *monitoring2.ClientMetricsPayload, err error) {
	q := "SELECT * FROM `measurements` as `metrics` WHERE `client_id` = ? "
	q, _ = query.ConvertOptionsToQuery(o, q)
	q = q + " LIMIT 1"

	val = new(monitoring2.ClientMetricsPayload)
	err = p.db.GetContext(ctx, val, q, clientID)
	return val, err
}

func (p *SqliteProvider) GetListByClientID(ctx context.Context, clientID string, o *query.Options) (val []monitoring2.ClientMetricsPayload, err error) {
	q := "SELECT * FROM `measurements` as `metrics` WHERE `client_id` = ? "
	q, _ = query.ConvertOptionsToQuery(o, q)

	val = []monitoring2.ClientMetricsPayload{}
	err = p.db.GetContext(ctx, &val, q, clientID)
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
