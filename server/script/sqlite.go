package script

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cloudradar-monitoring/rport/share/random"

	"github.com/cloudradar-monitoring/rport/db/migration/scripts"

	"github.com/cloudradar-monitoring/rport/share/query"

	chshare "github.com/cloudradar-monitoring/rport/share"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/sqlite"
)

type SqliteProvider struct {
	db     *sqlx.DB
	logger *chshare.Logger
}

var generateNewScriptID = func() (string, error) {
	return random.UUID4()
}

func NewSqliteProvider(dbPath string, logger *chshare.Logger) (*SqliteProvider, error) {
	db, err := sqlite.New(dbPath, scripts.AssetNames(), scripts.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed init scripts DB instance: %w", err)
	}

	logger.Infof("initialized database at %s", dbPath)

	return &SqliteProvider{logger: logger, db: db}, nil
}

func (p *SqliteProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}

	return nil
}

func (p *SqliteProvider) GetByID(ctx context.Context, id string, ro *query.RetrieveOptions) (val *Script, found bool, err error) {
	q := "SELECT * FROM `scripts` WHERE `id` = ? LIMIT 1"
	q = query.ConvertRetrieveOptionsToQuery(ro, q)

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

	q, params := query.ConvertListOptionsToQuery(lo, q)

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

		_, err = p.db.ExecContext(
			ctx,
			"INSERT INTO `scripts` (`id`, `name`, `created_at`, `created_by`, `interpreter`, `is_sudo`, `cwd`, `script`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			scriptID,
			s.Name,
			nowDate.Format(time.RFC3339),
			s.CreatedBy,
			s.Interpreter,
			s.IsSudo,
			s.Cwd,
			s.Script,
		)

		return scriptID, err
	}

	q := "UPDATE `scripts` SET `name` = ?, `interpreter` = ?, `is_sudo` = ?, `cwd` = ?, `script` = ? WHERE id = ?"
	params := []interface{}{
		s.Name,
		s.Interpreter,
		s.IsSudo,
		s.Cwd,
		s.Script,
		s.ID,
	}
	_, err := p.db.ExecContext(ctx, q, params...)

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
