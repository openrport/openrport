package auditlog

import (
	"context"
	"path"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/migration/auditlog"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type SQLiteProvider struct {
	db *sqlx.DB
}

func newSQLiteProvider(dataDir string, dataSourceOptions sqlite.DataSourceOptions) (*SQLiteProvider, error) {
	db, err := sqlite.New(
		path.Join(dataDir, sqliteFilename),
		auditlog.AssetNames(),
		auditlog.Asset,
		dataSourceOptions,
	)
	if err != nil {
		return nil, err
	}
	return &SQLiteProvider{
		db: db,
	}, nil
}

func (p *SQLiteProvider) Save(e *Entry) error {
	_, err := p.db.NamedExec(
		`INSERT INTO auditlog (
			timestamp,
			username,
			remote_ip,
			application,
			action,
			affected_id,
			client_id,
			client_hostname,
			request,
			response
		) VALUES (
			:timestamp,
			:username,
			:remote_ip,
			:application,
			:action,
			:affected_id,
			:client_id,
			:client_hostname,
			:request,
			:response
		)`,
		e,
	)

	return err
}

func (p *SQLiteProvider) List(ctx context.Context, options *query.ListOptions) ([]*Entry, error) {
	values := []*Entry{}

	q := "SELECT * FROM `auditlog`"

	q, params := query.ConvertListOptionsToQuery(options, q)

	err := p.db.SelectContext(ctx, &values, q, params...)
	if err != nil {
		return values, err
	}

	return values, nil
}

func (p *SQLiteProvider) Count(ctx context.Context, options *query.ListOptions) (int, error) {
	var result int

	q := "SELECT COUNT(*) FROM `auditlog`"
	countOptions := *options
	countOptions.Pagination = nil
	q, params := query.ConvertListOptionsToQuery(&countOptions, q)

	err := p.db.GetContext(ctx, &result, q, params...)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (p *SQLiteProvider) OldestTimestamp(ctx context.Context) (time.Time, error) {
	var ts time.Time
	q := "SELECT timestamp FROM auditlog ORDER BY timestamp ASC LIMIT 1"
	err := p.db.GetContext(ctx, &ts, q)
	if err != nil {
		return ts, err
	}
	return ts, nil
}

func (p *SQLiteProvider) Close() error {
	return p.db.Close()
}
