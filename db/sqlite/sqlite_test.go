package sqlite

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/db/migration/dummy"
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
