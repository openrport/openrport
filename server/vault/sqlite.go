package vault

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudradar-monitoring/rport/db/migration/vaults"
	chshare "github.com/cloudradar-monitoring/rport/share"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/sqlite"
)

type SqliteProvider struct {
	dbPath string
	db     *sqlx.DB
	logger *chshare.Logger
}

var ErrDatabaseNotInitialised = errors.New("vault is not initialized yet")

func NewSqliteProvider(c Config, logger *chshare.Logger) *SqliteProvider {
	dbPath := c.GetDatabasePath()
	if dbPath == "" {
		dbPath = defaultDBName
	}

	return &SqliteProvider{dbPath: dbPath, logger: logger}
}

func (p *SqliteProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}

	return nil
}

func (p *SqliteProvider) Init(ctx context.Context) error {
	db, err := sqlite.New(p.dbPath, vaults.AssetNames(), vaults.Asset)
	if err != nil {
		return fmt.Errorf("failed init vault DB instance: %w", err)
	}
	p.logger.Infof("initialized database at %s", p.dbPath)

	p.db = db

	return nil
}

func (p *SqliteProvider) GetStatus(ctx context.Context) (DbStatus, error) {
	db, err := p.getDb()
	if err != nil {
		return DbStatus{}, err
	}

	res := DbStatus{}
	err = db.GetContext(ctx, &res, "SELECT * FROM `status` LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return res, err
	}

	return res, nil
}

func (p *SqliteProvider) SetStatus(ctx context.Context, newStatus DbStatus) error {
	db, err := p.getDb()
	if err != nil {
		return err
	}

	tx, err := db.Beginx()
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
	db, err := p.getDb()
	if err != nil {
		return val, found, err
	}

	err = db.GetContext(ctx, &val, "SELECT * FROM `values` WHERE `id` = ? LIMIT 1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return val, false, nil
		}

		return val, false, err
	}

	return val, true, nil
}

func (p *SqliteProvider) List(ctx context.Context, lo *ListOptions) ([]ValueKey, error) {
	values := []ValueKey{}
	db, err := p.getDb()
	if err != nil {
		return values, err
	}

	q := "SELECT `id`, `client_id`, `created_by`, `created_at`, `key` FROM `values`"

	q, params := p.addWhere(lo, q)
	q = p.addOrderBy(lo, q)

	err = db.SelectContext(ctx, &values, q, params...)
	if err != nil {
		return values, err
	}

	return values, nil
}

func (p *SqliteProvider) addWhere(lo *ListOptions, q string) (qOut string, params []interface{}) {
	params = []interface{}{}
	if len(lo.Filters) == 0 {
		return q, params
	}

	whereParts := make([]string, 0, len(lo.Filters))
	for i := range lo.Filters {
		if len(lo.Filters[i].Values) == 1 {
			whereParts = append(whereParts, fmt.Sprintf("%s = ?", lo.Filters[i].Column))
			params = append(params, lo.Filters[i].Values[0])
		} else {
			orParts := make([]string, 0, len(lo.Filters[i].Values))
			for y := range lo.Filters[i].Values {
				orParts = append(orParts, fmt.Sprintf("%s = ?", lo.Filters[i].Column))
				params = append(params, lo.Filters[i].Values[y])
			}

			whereParts = append(whereParts, fmt.Sprintf("(%s)", strings.Join(orParts, " OR ")))
		}
	}

	q += " WHERE " + strings.Join(whereParts, " AND ")

	return q, params
}

func (p *SqliteProvider) addOrderBy(lo *ListOptions, q string) string {
	if len(lo.Sorts) == 0 {
		return q
	}
	orderByValues := make([]string, 0, len(lo.Sorts))
	for i := range lo.Sorts {
		direction := "ASC"
		if !lo.Sorts[i].IsASC {
			direction = "DESC"
		}
		orderByValues = append(orderByValues, fmt.Sprintf("%s %s", lo.Sorts[i].Column, direction))
	}
	if len(orderByValues) > 0 {
		q += "ORDER BY " + strings.Join(orderByValues, ",")
	}

	return q
}

func (p *SqliteProvider) FindByKeyAndClientID(ctx context.Context, key, clientID string) (val StoredValue, found bool, err error) {
	db, err := p.getDb()
	if err != nil {
		return val, found, err
	}

	err = db.GetContext(ctx, &val, "SELECT * FROM `values` WHERE `key` = ? and `client_id` = ? LIMIT 1", key, clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return val, false, nil
		}

		return val, false, err
	}

	return val, true, nil
}

func (p *SqliteProvider) Save(ctx context.Context, user string, idToUpdate int64, val *InputValue, nowDate time.Time) (int64, error) {
	db, err := p.getDb()
	if err != nil {
		return 0, err
	}

	if idToUpdate == 0 {
		res, err := db.ExecContext(
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
		_, err = db.ExecContext(ctx, q, params...)
		if err != nil {
			return 0, err
		}
	}

	return idToUpdate, nil
}

func (p *SqliteProvider) Delete(ctx context.Context, id int) error {
	db, err := p.getDb()
	if err != nil {
		return err
	}

	res, err := db.ExecContext(ctx, "DELETE FROM `values` WHERE `id` = ?", id)

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

func (p *SqliteProvider) getDb() (*sqlx.DB, error) {
	if p.db == nil {
		return nil, ErrDatabaseNotInitialised
	}

	return p.db, nil
}

func (p *SqliteProvider) handleRollback(tx *sqlx.Tx) {
	err := tx.Rollback()
	if err != nil {
		p.logger.Errorf("Failed to rollback transaction: %v", err)
	}
}
