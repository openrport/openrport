package auditlog

import (
	"context"
	"path"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/migration/auditlog"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type SQLiteProvider struct {
	db *sqlx.DB
}

func newSQLiteProvider(dataDir string) (*SQLiteProvider, error) {
	db, err := sqlite.New(path.Join(dataDir, "auditlog.db"), auditlog.AssetNames(), auditlog.Asset)
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

func (p *SQLiteProvider) Close() error {
	return p.db.Close()
}
