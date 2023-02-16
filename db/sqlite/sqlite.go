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
	DefaultMaxOpenConnections   = 1
)

type DataSourceOptions struct {
	WALEnabled         bool
	MaxOpenConnections int
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

	maxConns := dataSourceOptions.MaxOpenConnections
	if maxConns == 0 {
		maxConns = DefaultMaxOpenConnections
	}

	db.SetMaxOpenConns(maxConns)

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

func WithRetryWhenBusy[R any](retryAbleFn func() (result R, err error), label string, l *logger.Logger) (result R, err error) {
	for attempt := 1; attempt <= DefaultMaxAttempts; attempt++ {
		if attempt > 1 && err != nil {
			sqlErr, ok := err.(sql.Error)
			if ok && sqlErr.Code == sql.ErrBusy {
				l.Debugf("%s: attempt %d: source err = %+v\n", label, attempt, err)
				jitter := time.Duration((rand.Intn(100))) * time.Millisecond
				time.Sleep(defaultDelayBetweenAttempts + jitter)
			} else {
				// a different error from database busy, so fail immediately
				l.Debugf("%s: attempt %d: non-retryable err = %+v\n", label, attempt, err)
				return result, err
			}
		}
		// make an attempt to complete the retryable fn
		result, err = retryAbleFn()
		// if no error then return immediately if with success result
		if err == nil {
			return result, nil
		}
	}

	l.Errorf("%s: failed after max attempts: err = %+v\n", label, err)
	return result, err
}
