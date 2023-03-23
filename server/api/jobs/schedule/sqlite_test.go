package schedule

import (
	"context"
	"os"
	"testing"
	"time"

	jobsmigration "github.com/realvnc-labs/rport/db/migration/jobs"
	"github.com/realvnc-labs/rport/db/sqlite"
	"github.com/realvnc-labs/rport/server/api/jobs"
	"github.com/realvnc-labs/rport/server/test/jb"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/ptr"

	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}

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
	db, err := sqlite.New(":memory:", jobsmigration.AssetNames(), jobsmigration.Asset, DataSourceOptions)
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
	db, err := sqlite.New(":memory:", jobsmigration.AssetNames(), jobsmigration.Asset, DataSourceOptions)
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
	db, err := sqlite.New(":memory:", jobsmigration.AssetNames(), jobsmigration.Asset, DataSourceOptions)
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
	db, err := sqlite.New(":memory:", jobsmigration.AssetNames(), jobsmigration.Asset, DataSourceOptions)
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
	db, err := sqlite.New(":memory:", jobsmigration.AssetNames(), jobsmigration.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := newSQLiteProvider(db)
	defer dbProv.Close()
	ctx := context.Background()

	err = addTestData(dbProv.db)
	require.NoError(t, err)
	addJobs(t, db)

	err = dbProv.Delete(ctx, "-2")
	assert.EqualError(t, err, "cannot find entry by id -2")

	err = dbProv.Delete(ctx, "2")
	require.NoError(t, err)

	val, err := dbProv.Get(ctx, "2")
	require.NoError(t, err)
	assert.Nil(t, val)

	assertCount(t, db, 3, "SELECT count(*) FROM jobs")
	assertCount(t, db, 1, "SELECT count(*) FROM multi_jobs WHERE schedule_id = '1'")
	assertCount(t, db, 0, "SELECT count(*) FROM multi_jobs WHERE schedule_id = '2'")
}

func TestCountJobsInProgress(t *testing.T) {
	db, err := sqlite.New(":memory:", jobsmigration.AssetNames(), jobsmigration.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := newSQLiteProvider(db)
	defer dbProv.Close()
	ctx := context.Background()

	addJobs(t, db)

	count, err := dbProv.CountJobsInProgress(ctx, testData[0].ID, 60)
	require.NoError(t, err)

	// Counted only the non finished job
	assert.Equal(t, 1, count)
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

func addJobs(t *testing.T, db *sqlx.DB) {
	testLog := logger.NewLogger("test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	jobsProvider := jobs.NewSqliteProvider(db, testLog)

	multiJob := jb.NewMulti(t).ScheduleID(testData[0].ID).Build()
	otherMultiJob := jb.NewMulti(t).ScheduleID(testData[1].ID).Build()
	require.NoError(t, jobsProvider.SaveMultiJob(multiJob))
	require.NoError(t, jobsProvider.SaveMultiJob(otherMultiJob))

	// finished job
	job1 := jb.New(t).MultiJobID(multiJob.JID).FinishedAt(time.Now()).Build()
	// non finished job
	job2 := jb.New(t).MultiJobID(multiJob.JID).StartedAt(time.Now()).Build()
	// non finished expired job
	job3 := jb.New(t).MultiJobID(multiJob.JID).StartedAt(time.Now().Add(-2 * time.Minute)).Build()
	// from other schedule
	job4 := jb.New(t).MultiJobID(otherMultiJob.JID).Build()
	require.NoError(t, jobsProvider.SaveJob(job1))
	require.NoError(t, jobsProvider.SaveJob(job2))
	require.NoError(t, jobsProvider.SaveJob(job3))
	require.NoError(t, jobsProvider.SaveJob(job4))
}

func assertCount(t *testing.T, db *sqlx.DB, expected int, query string) {
	t.Helper()

	var result int
	err := db.Get(&result, query)
	require.NoError(t, err)

	assert.Equal(t, expected, result)
}
