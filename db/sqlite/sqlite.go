package sqlite

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/jmoiron/sqlx"
	sql "github.com/mattn/go-sqlite3"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

const (
	WALEnabled                  = "_journal_mode=WAL"
	defaultDelayBetweenAttempts = 200 * time.Millisecond
	DefaultMaxAttempts          = 5
)

type DataSourceOptions struct {
	WALEnabled bool
}

// New returns a new sqlite DB instance with migrated DB scheme to the latest version.
// assetNames and asset are used to migrate DB scheme.
func New(dataSourceName string, assetNames []string, asset func(name string) ([]byte, error), dataSourceOptions DataSourceOptions) (*sqlx.DB, error) {
	dbPath := dataSourceName
	if dataSourceOptions.WALEnabled {
		dataSourceName += "?" + WALEnabled
	}
	db, err := sqlx.Connect("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %v", err)
	}
	if dbPath != ":memory:" {
		if err = os.Chmod(dbPath, 0600); err != nil {
			return nil, fmt.Errorf("failed to chmod %s: %s", dbPath, err)
		}
	}

	db.SetMaxOpenConns(1)

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

// TODO: (rs): we've moved to use single db connections. with potentially slower access to the sqlite
// volumes it seems there's too much concurrent contention for the dbs, so there's less need for this fn.
// not removing yet but check again in approx 6 months (from dec 22) and remove if no longer required.
func WithRetryWhenBusy[R any](retryAble func() (result R, err error), label string, l *logger.Logger) (result R, err error) {
	for r := 0; r < DefaultMaxAttempts; r++ {
		result, err = retryAble()
		if err != nil {
			sqlErr, ok := err.(sql.Error)
			if ok && sqlErr.Code == sql.ErrBusy {
				l.Debugf("%s: attempt %d: source err = %+v\n", label, r+1, err)
				jitter := time.Duration((rand.Intn(100))) * time.Millisecond
				time.Sleep(defaultDelayBetweenAttempts + jitter)
				continue
			}
			// non retryable err
			return result, sqlErr
		}
		// success
		return result, nil
	}

	l.Debugf("%s: failed after max attempts: err = %+v\n", label, err)
	return result, err
}
