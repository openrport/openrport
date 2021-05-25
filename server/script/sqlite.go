package script

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

func (p *SqliteProvider) GetByID(ctx context.Context, id int64) (val *Script, found bool, err error) {
	val = new(Script)
	err = p.db.GetContext(ctx, val, "SELECT * FROM `scripts` WHERE `id` = ? LIMIT 1", id)
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

	q := "SELECT *  FROM `scripts`"

	q, params := query.ConvertListOptionsToQuery(lo, q)

	err := p.db.SelectContext(ctx, &values, q, params...)
	if err != nil {
		return values, err
	}

	return values, nil
}

func (p *SqliteProvider) Save(ctx context.Context, s *Script, nowDate time.Time) (int64, error) {
	if s.ID == 0 {
		res, err := p.db.ExecContext(
			ctx,
			"INSERT INTO `scripts` (`name`, `created_at`, `created_by`, `interpreter`, `is_sudo`, `cwd`, `script`) VALUES (?, ?, ?, ?, ?, ?, ?)",
			s.Name,
			nowDate.Format(time.RFC3339),
			s.CreatedBy,
			s.Interpreter,
			s.IsSudo,
			s.Cwd,
			s.Script,
		)

		if err != nil {
			return 0, err
		}
		s.ID, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else {
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
		if err != nil {
			return 0, err
		}
	}

	return s.ID, nil
}

func (p *SqliteProvider) Delete(ctx context.Context, id int64) error {
	res, err := p.db.ExecContext(ctx, "DELETE FROM `scripts` WHERE `id` = ?", id)

	if err != nil {
		return err
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affectedRows == 0 {
		return fmt.Errorf("cannot find entry by id %d", id)
	}

	return nil
}
