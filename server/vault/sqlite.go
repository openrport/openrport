package vault

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
