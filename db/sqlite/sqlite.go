package sqlite

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/jmoiron/sqlx"
)

const WALEnabled = "_journal_mode=WAL"

type DataSourceOptions struct {
	WALEnabled bool
}

// New returns a new sqlite DB instance with migrated DB scheme to the latest version.
// assetNames and asset are used to migrate DB scheme.
func New(dataSourceName string, assetNames []string, asset func(name string) ([]byte, error), dataSourceOptions DataSourceOptions) (*sqlx.DB, error) {
	if dataSourceOptions.WALEnabled {
		dataSourceName += "?" + WALEnabled
	}
	db, err := sqlx.Connect("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %v", err)
	}

	s := bindata.Resource(assetNames,
		func(name string) ([]byte, error) {
			return asset(name)
		})
	sourceDriver, err := bindata.WithInstance(s)
	if err != nil {
		return nil, fmt.Errorf("failed to init DB source driver: %v", err)
	}

	dbDriver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to init DB migration driver: %v", err)
	}

	m, err := migrate.NewWithInstance("go-bindata", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return nil, fmt.Errorf("failed to init DB migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("failed to migrate DB to the latest version: %v", err)
	}

	return db, nil
}
