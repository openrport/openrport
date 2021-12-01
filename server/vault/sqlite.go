package vault

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/query"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/migration/vaults"

	"github.com/cloudradar-monitoring/rport/db/sqlite"
)

var ErrDatabaseNotInitialised = errors2.APIError{
	Err:        errors.New("vault is not initialized yet"),
	HTTPStatus: http.StatusConflict,
}

type SqliteProvider struct {
	db     *sqlx.DB
	logger *logger.Logger
}

func NewSqliteProvider(c Config, logger *logger.Logger) (*SqliteProvider, error) {
	dbPath := c.GetVaultDBPath()

	db, err := sqlite.New(dbPath, vaults.AssetNames(), vaults.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed init vault DB instance: %w", err)
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

func (p *SqliteProvider) GetStatus(ctx context.Context) (DbStatus, error) {
	res := DbStatus{}
	err := p.db.GetContext(ctx, &res, "SELECT * FROM `status` LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return res, err
	}

	return res, nil
}

func (p *SqliteProvider) SetStatus(ctx context.Context, newStatus DbStatus) error {
	tx, err := p.db.Beginx()
	if err != nil {
		return err
	}

	var idToUpdate int
	err = tx.GetContext(ctx, &idToUpdate, "SELECT id FROM `status` LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			idToUpdate = 0
		} else {
			p.handleRollback(tx)
			return err
		}
	}

	if idToUpdate == 0 {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO `status` (`db_status`, `enc_check`, `dec_check`) VALUES (?, ?, ?)",
			newStatus.StatusName,
			newStatus.EncCheckValue,
			newStatus.DecCheckValue,
		)

		if err != nil {
			p.handleRollback(tx)
			return err
		}
	} else {
		q := "UPDATE `status` SET db_status=?, enc_check = ?, dec_check = ? WHERE id = ?"
		params := []interface{}{
			newStatus.StatusName,
			newStatus.EncCheckValue,
			newStatus.DecCheckValue,
			idToUpdate,
		}
		_, err = tx.ExecContext(ctx, q, params...)
		if err != nil {
			p.handleRollback(tx)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (p *SqliteProvider) GetByID(ctx context.Context, id int) (val StoredValue, found bool, err error) {
	err = p.db.GetContext(ctx, &val, "SELECT * FROM `values` WHERE `id` = ? LIMIT 1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return val, false, nil
		}

		return val, false, err
	}

	return val, true, nil
}

func (p *SqliteProvider) List(ctx context.Context, lo *query.ListOptions) ([]ValueKey, error) {
	values := []ValueKey{}

	q := "SELECT `id`, `client_id`, `created_by`, `created_at`, `key` FROM `values`"

	q, params := query.ConvertListOptionsToQuery(lo, q)

	err := p.db.SelectContext(ctx, &values, q, params...)
	if err != nil {
		return values, err
	}

	return values, nil
}

func (p *SqliteProvider) FindByKeyAndClientID(ctx context.Context, key, clientID string) (val StoredValue, found bool, err error) {
	err = p.db.GetContext(ctx, &val, "SELECT * FROM `values` WHERE `key` = ? and `client_id` = ? LIMIT 1", key, clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return val, false, nil
		}

		return val, false, err
	}

	return val, true, nil
}

func (p *SqliteProvider) Save(ctx context.Context, user string, idToUpdate int64, val *InputValue, nowDate time.Time) (int64, error) {
	if idToUpdate == 0 {
		res, err := p.db.ExecContext(
			ctx,
			"INSERT INTO `values` (`client_id`, `required_group`, `created_at`, `created_by`, `updated_at`, `updated_by`, `key`, `value`, `type`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			val.ClientID,
			val.RequiredGroup,
			nowDate.Format(time.RFC3339),
			user,
			nowDate.Format(time.RFC3339),
			user,
			val.Key,
			val.Value,
			val.Type,
		)

		if err != nil {
			return 0, err
		}
		idToUpdate, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else {
		q := "UPDATE `values` SET `client_id` = ?, `required_group` = ?, `updated_at` = ?, `updated_by` = ?, `key` = ?, `value` = ?, `type` = ? WHERE id = ?"
		params := []interface{}{
			val.ClientID,
			val.RequiredGroup,
			nowDate.Format(time.RFC3339),
			user,
			val.Key,
			val.Value,
			val.Type,
			idToUpdate,
		}
		_, err := p.db.ExecContext(ctx, q, params...)
		if err != nil {
			return 0, err
		}
	}

	return idToUpdate, nil
}

func (p *SqliteProvider) Delete(ctx context.Context, id int) error {
	res, err := p.db.ExecContext(ctx, "DELETE FROM `values` WHERE `id` = ?", id)

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

func (p *SqliteProvider) handleRollback(tx *sqlx.Tx) {
	err := tx.Rollback()
	if err != nil {
		p.logger.Errorf("Failed to rollback transaction: %v", err)
	}
}

type NotInitDbProvider struct{}

func (nidp *NotInitDbProvider) Init(ctx context.Context) error {
	return ErrDatabaseNotInitialised
}

func (nidp *NotInitDbProvider) GetStatus(ctx context.Context) (DbStatus, error) {
	return DbStatus{}, ErrDatabaseNotInitialised
}

func (nidp *NotInitDbProvider) SetStatus(ctx context.Context, newStatus DbStatus) error {
	return ErrDatabaseNotInitialised
}

func (nidp *NotInitDbProvider) GetByID(ctx context.Context, id int) (val StoredValue, found bool, err error) {
	err = ErrDatabaseNotInitialised
	return
}

func (nidp *NotInitDbProvider) List(ctx context.Context, lo *query.ListOptions) ([]ValueKey, error) {
	return nil, ErrDatabaseNotInitialised
}

func (nidp *NotInitDbProvider) FindByKeyAndClientID(ctx context.Context, key, clientID string) (val StoredValue, found bool, err error) {
	err = ErrDatabaseNotInitialised
	return
}

func (nidp *NotInitDbProvider) Save(ctx context.Context, user string, idToUpdate int64, val *InputValue, nowDate time.Time) (int64, error) {
	return 0, ErrDatabaseNotInitialised
}

func (nidp *NotInitDbProvider) Delete(ctx context.Context, id int) error {
	return ErrDatabaseNotInitialised
}

func (nidp *NotInitDbProvider) Close() error {
	return nil
}
