package auditlog

import (
	"path"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/migration/auditlog"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
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

func (p *SQLiteProvider) Close() error {
	return p.db.Close()
}
