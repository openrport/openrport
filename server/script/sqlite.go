package script

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/realvnc-labs/rport/share/random"

	"github.com/realvnc-labs/rport/share/query"

	"github.com/jmoiron/sqlx"
)

type SqliteProvider struct {
	db        *sqlx.DB
	converter *query.SQLConverter
}

var generateNewScriptID = func() (string, error) {
	return random.UUID4()
}

func NewSqliteProvider(db *sqlx.DB) *SqliteProvider {
	return &SqliteProvider{
		db:        db,
		converter: query.NewSQLConverter(db.DriverName()),
	}
}

func (p *SqliteProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}

	return nil
}

func (p *SqliteProvider) GetByID(ctx context.Context, id string, ro *query.RetrieveOptions) (val *Script, found bool, err error) {
	q := "SELECT * FROM `scripts` WHERE `id` = ? LIMIT 1"
	q = p.converter.ConvertRetrieveOptionsToQuery(ro, q)

	val = new(Script)
	err = p.db.GetContext(ctx, val, q, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return val, false, nil
		}

		return val, false, err
	}

	return val, true, nil
}

func (p *SqliteProvider) List(ctx context.Context, lo *query.ListOptions) ([]Script, error) {
	values := []Script{}

	q := "SELECT * FROM `scripts`"

	q, params := p.converter.ConvertListOptionsToQuery(lo, q)

	err := p.db.SelectContext(ctx, &values, q, params...)
	if err != nil {
		return values, err
	}

	return values, nil
}

func (p *SqliteProvider) Save(ctx context.Context, s *Script, nowDate time.Time) (string, error) {
	if s.ID == "" {
		scriptID, err := generateNewScriptID()
		if err != nil {
			return scriptID, err
		}
		s.ID = scriptID

		_, err = p.db.NamedExecContext(
			ctx,
			"INSERT INTO `scripts`"+
				" (`id`, `name`, `created_at`, `created_by`, `interpreter`, `is_sudo`, `cwd`, `script`, `updated_at`, `updated_by`, `tags`, `timeout_sec`)"+
				" VALUES "+
				"(:id, :name, :created_at, :created_by, :interpreter, :is_sudo, :cwd, :script, :updated_at, :updated_by, :tags, :timeout_sec)",
			s,
		)

		return scriptID, err
	}

	q := "UPDATE `scripts` SET " +
		"`name` = :name, " +
		"`interpreter` = :interpreter, " +
		"`is_sudo` = :is_sudo, " +
		"`cwd` = :cwd, " +
		"`script` = :script, " +
		"`updated_at` = :updated_at, " +
		"`updated_by` = :updated_by, " +
		"`tags` = :tags, " +
		"`timeout_sec` = :timeout_sec" +
		" WHERE id = :id "

	_, err := p.db.NamedExecContext(ctx, q, s)

	return s.ID, err
}

func (p *SqliteProvider) Delete(ctx context.Context, id string) error {
	res, err := p.db.ExecContext(ctx, "DELETE FROM `scripts` WHERE `id` = ?", id)

	if err != nil {
		return err
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affectedRows == 0 {
		return fmt.Errorf("cannot find entry by id %s", id)
	}

	return nil
}
