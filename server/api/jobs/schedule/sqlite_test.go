package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/db/migration/jobs"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/ptr"

	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testData = []*Schedule{
	{
		Base: Base{
			ID:        "1",
			Name:      "schedule 1",
			CreatedBy: "user1",
			CreatedAt: time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
			Schedule:  "* * * * *",
			Type:      "command",
		},
		Details: Details{
			ClientIDs: []string{"c1"},
			GroupIDs:  []string{"g1"},
			Command:   "/bin/true",
			Cwd:       "/home/rport",
			Overlaps:  true,
		},
	},
	{
		Base: Base{
			ID:        "2",
			Name:      "schedule 2",
			CreatedBy: "user1",
			CreatedAt: time.Date(2022, 1, 1, 2, 0, 0, 0, time.UTC),
			Schedule:  "*/5 * * * *",
			Type:      "script",
		},
		Details: Details{
			ClientIDs:           []string{"c2"},
			GroupIDs:            []string{"g2"},
			Command:             "echo 'test'",
			Interpreter:         "/bin/sh",
			Cwd:                 "/home/rport",
			TimeoutSec:          3,
			ExecuteConcurrently: true,
			AbortOnError:        ptr.Bool(true),
			Overlaps:            true,
		},
	},
}

func TestGet(t *testing.T) {
	db, err := sqlite.New(":memory:", jobs.AssetNames(), jobs.Asset)
	require.NoError(t, err)
	dbProv := newSQLiteProvider(db)
	defer dbProv.Close()
	ctx := context.Background()

	err = addTestData(dbProv.db)
	require.NoError(t, err)

	val, err := dbProv.Get(ctx, "1")
	require.NoError(t, err)
	assert.Equal(t, testData[0], val)

	val, err = dbProv.Get(ctx, "-2")
	require.NoError(t, err)
	assert.Nil(t, val)
}

func TestList(t *testing.T) {
	db, err := sqlite.New(":memory:", jobs.AssetNames(), jobs.Asset)
	require.NoError(t, err)
	dbProv := newSQLiteProvider(db)
	defer dbProv.Close()

	err = addTestData(dbProv.db)
	require.NoError(t, err)

	result, err := dbProv.List(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, testData, result)
}

func TestCreate(t *testing.T) {
	db, err := sqlite.New(":memory:", jobs.AssetNames(), jobs.Asset)
	require.NoError(t, err)
	dbProv := newSQLiteProvider(db)
	defer dbProv.Close()
	ctx := context.Background()

	itemToSave := testData[0]
	err = dbProv.Insert(ctx, itemToSave)
	require.NoError(t, err)

	val, err := dbProv.Get(ctx, "1")
	require.NoError(t, err)
	assert.Equal(t, testData[0], val)
}

func TestUpdate(t *testing.T) {
	db, err := sqlite.New(":memory:", jobs.AssetNames(), jobs.Asset)
	require.NoError(t, err)
	dbProv := newSQLiteProvider(db)
	defer dbProv.Close()
	ctx := context.Background()

	err = addTestData(dbProv.db)
	require.NoError(t, err)

	itemToUpdate := *testData[0]
	itemToUpdate.Schedule = "1 5 * * *"

	err = dbProv.Update(ctx, &itemToUpdate)
	require.NoError(t, err)

	val, err := dbProv.Get(ctx, "1")
	require.NoError(t, err)
	assert.NotEqual(t, testData[0], val)
	assert.Equal(t, &itemToUpdate, val)
}

func TestDelete(t *testing.T) {
	db, err := sqlite.New(":memory:", jobs.AssetNames(), jobs.Asset)
	require.NoError(t, err)
	dbProv := newSQLiteProvider(db)
	defer dbProv.Close()
	ctx := context.Background()

	err = addTestData(dbProv.db)
	require.NoError(t, err)

	err = dbProv.Delete(ctx, "-2")
	assert.EqualError(t, err, "cannot find entry by id -2")

	err = dbProv.Delete(ctx, "2")
	require.NoError(t, err)

	val, err := dbProv.Get(ctx, "2")
	require.NoError(t, err)
	assert.Nil(t, val)
}

func addTestData(db *sqlx.DB) error {
	for _, row := range testData {
		_, err := db.Exec(
			"INSERT INTO `schedules` (`id`, `name`, `created_at`, `created_by`, `schedule`, `type`, `details`) VALUES (?,?,?,?,?,?,?)",
			row.ID,
			row.Name,
			row.CreatedAt,
			row.CreatedBy,
			row.Schedule,
			row.Type,
			row.Details,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
