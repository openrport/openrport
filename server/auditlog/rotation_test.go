package auditlog

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/db/migration/auditlog"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/query"
)

func TestRotation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	period := 300 * time.Millisecond

	// Prepare sqlite with 1 entry
	sqlite, err := newSQLiteProvider(dir)
	require.NoError(t, err)
	err = sqlite.Save(&Entry{Timestamp: time.Now(), Username: "test1"})
	require.NoError(t, err)
	err = sqlite.Close()
	require.NoError(t, err)

	// No rotation on init if entry is not older than period
	rotation, err := newRotationProvider(nil, period, dir)
	require.NoError(t, err)
	entries, err := rotation.List(ctx, &query.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, "test1", entries[0].Username)
	err = rotation.Close()
	require.NoError(t, err)

	time.Sleep(period)

	// Should rotate on init
	rotation, err = newRotationProvider(nil, period, dir)
	require.NoError(t, err)
	entries, err = rotation.List(ctx, &query.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(entries))
	assertRotatedSqlite(t, dir, "test1")

	// Add entry
	err = rotation.Save(&Entry{Timestamp: time.Now(), Username: "test2"})
	require.NoError(t, err)

	// Read back entry
	entries, err = rotation.List(ctx, &query.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, "test2", entries[0].Username)

	time.Sleep(period)

	// Should be rotated after period
	entries, err = rotation.List(ctx, &query.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(entries))
	err = rotation.Close()
	require.NoError(t, err)
	assertRotatedSqlite(t, dir, "test2")
}

func assertRotatedSqlite(t *testing.T, dir, expectedUsername string) {
	db, err := sqlite.New(path.Join(dir, time.Now().Format(rotatedFilename)), auditlog.AssetNames(), auditlog.Asset)
	require.NoError(t, err)
	sqlite := &SQLiteProvider{
		db: db,
	}
	defer sqlite.Close()

	entries, err := sqlite.List(context.Background(), &query.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, expectedUsername, entries[0].Username)
}
