package sqlite

import (
	"os"
	"testing"

	sql "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/db/migration/dummy"
	"github.com/realvnc-labs/rport/share/logger"
)

func TestSqliteWALEnabled(t *testing.T) {
	dataSourceName := t.TempDir() + "/test-db.sqlite3"
	_, err := New(dataSourceName, dummy.AssetNames(), dummy.Asset, DataSourceOptions{WALEnabled: true})
	require.NoError(t, err)
	_, err = os.Stat(dataSourceName + "-shm")
	require.NoError(t, err)
	_, err = os.Stat(dataSourceName + "-wal")
	require.NoError(t, err)
}

func TestSqliteWALDisabled(t *testing.T) {
	dataSourceName := t.TempDir() + "/test-db.sqlite3"
	_, err := New(dataSourceName, dummy.AssetNames(), dummy.Asset, DataSourceOptions{WALEnabled: false})
	require.NoError(t, err)
	_, err = os.Stat(dataSourceName + "-shm")
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(dataSourceName + "-wal")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestShouldSucceedWhenNoError(t *testing.T) {
	testLog := logger.NewLogger("retries", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	_, err := WithRetryWhenBusy(func() (result any, err error) {
		return nil, nil
	}, "test", testLog)

	assert.NoError(t, err)
}

func TestShouldSucceedAfterRetries(t *testing.T) {
	testLog := logger.NewLogger("retries", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	attempts := 0

	_, err := WithRetryWhenBusy(func() (result any, err error) {
		if attempts < 2 {
			attempts++
			return nil, sql.Error{
				Code: sql.ErrBusy,
			}
		}
		return nil, nil
	}, "test", testLog)

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestShouldFailWhenMaxBusyErrors(t *testing.T) {
	testLog := logger.NewLogger("retries", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	attempts := 0

	_, err := WithRetryWhenBusy(func() (result any, err error) {
		attempts++
		return nil, sql.Error{
			Code: sql.ErrBusy,
		}
	}, "test", testLog)

	assert.EqualError(t, err, sql.ErrBusy.Error())
	assert.Equal(t, DefaultMaxAttempts, attempts)
}

func TestShouldFailImmediatelyWhenNonBusyError(t *testing.T) {
	testLog := logger.NewLogger("retries", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	attempts := 0

	_, err := WithRetryWhenBusy(func() (result any, err error) {
		attempts++
		// fail immediately when not sql.ErrBusy
		return nil, sql.Error{
			Code: sql.ErrCorrupt,
		}
	}, "test", testLog)

	assert.EqualError(t, err, sql.ErrCorrupt.Error())
	assert.Equal(t, 1, attempts)
}
