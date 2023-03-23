package command

import (
	"context"
	"testing"
	"time"

	"github.com/realvnc-labs/rport/db/migration/library"
	"github.com/realvnc-labs/rport/db/sqlite"
	"github.com/realvnc-labs/rport/share/ptr"
	"github.com/realvnc-labs/rport/share/query"

	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/share/test"
)

var timeoutSec = DefaultTimeoutSec
var demoData = []Command{
	{
		ID:        "1",
		Name:      "some name",
		CreatedBy: "user1",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		UpdatedBy: "user2",
		UpdatedAt: ptr.Time(time.Date(2003, 1, 1, 1, 0, 0, 0, time.UTC)),
		Cmd:       "ls -la",
		Tags:      ptr.StringSlice("tag1", "tag2"),
		TimoutSec: &timeoutSec,
	},
	{
		ID:        "2",
		Name:      "other name 2",
		CreatedBy: "user1",
		CreatedAt: ptr.Time(time.Date(2002, 1, 1, 1, 0, 0, 0, time.UTC)),
		UpdatedBy: "user1",
		UpdatedAt: ptr.Time(time.Date(2002, 1, 1, 2, 0, 0, 0, time.UTC)),
		Cmd:       "pwd",
		Tags:      ptr.StringSlice(),
		TimoutSec: &timeoutSec,
	},
}
var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}

func TestGetByID(t *testing.T) {
	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()
	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	var found bool
	var com *Command

	t.Run("first test", func(t *testing.T) {
		com, found, err = dbProv.GetByID(ctx, "1", &query.RetrieveOptions{})
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, demoData[0], *com)
		_, found, err = dbProv.GetByID(ctx, "-2", &query.RetrieveOptions{})
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("second test", func(t *testing.T) {
		com, found, err = dbProv.GetByID(ctx, "1", &query.RetrieveOptions{Fields: []query.FieldsOption{
			{
				Resource: "commands",
				Fields:   []string{"created_by", "cmd"},
			},
		}})
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, Command{
			CreatedBy: "user1",
			Cmd:       "ls -la",
		}, *com)
	})
}

func TestList(t *testing.T) {
	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	t.Cleanup(func() { dbProv.Close() })

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	testCases := []struct {
		Name           string
		Options        *query.ListOptions
		ExpectedResult []Command
	}{
		{
			Name:           "no options",
			Options:        &query.ListOptions{},
			ExpectedResult: demoData,
		},
		{
			Name: "sort only",
			Options: &query.ListOptions{
				Sorts: []query.SortOption{
					{
						Column: "created_at",
						IsASC:  false,
					},
				},
			},
			ExpectedResult: []Command{demoData[1], demoData[0]},
		},
		{
			Name: "filter and sort",
			Options: &query.ListOptions{
				Sorts: []query.SortOption{
					{
						Column: "name",
						IsASC:  true,
					},
				},
				Filters: []query.FilterOption{
					{
						Column: []string{"created_by"},
						Values: []string{"user1"},
					},
				},
			},
			ExpectedResult: []Command{demoData[1], demoData[0]},
		},
		{
			Name: "filter, no results",
			Options: &query.ListOptions{
				Filters: []query.FilterOption{
					{
						Column: []string{"name"},
						Values: []string{"not-existing-name"},
					},
				},
			},
			ExpectedResult: []Command{},
		},
		{
			Name: "filter, 1 result",
			Options: &query.ListOptions{
				Filters: []query.FilterOption{
					{
						Column: []string{"name"},
						Values: []string{"some name"},
					},
				},
			},
			ExpectedResult: []Command{demoData[0]},
		},
		{
			Name: "multiple filters",
			Options: &query.ListOptions{
				Sorts: []query.SortOption{
					{
						Column: "created_at",
						IsASC:  true,
					},
				},
				Filters: []query.FilterOption{
					{
						Column: []string{"name"},
						Values: []string{"some name", "other name 2"},
					},
					{
						Column: []string{"created_by"},
						Values: []string{"user1"},
					},
				},
			},
			ExpectedResult: demoData,
		},
		{
			Name: "fields",
			Options: &query.ListOptions{
				Fields: []query.FieldsOption{
					{
						Resource: "commands",
						Fields:   []string{"id", "name"},
					},
				},
			},
			ExpectedResult: []Command{
				{
					ID:   "1",
					Name: "some name",
				},
				{
					ID:   "2",
					Name: "other name 2",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			result, err := dbProv.List(context.Background(), tc.Options)
			require.NoError(t, err)

			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}

func TestCreate(t *testing.T) {
	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()
	ctx := context.Background()

	itemToSave := demoData[0]
	itemToSave.ID = ""
	id, err := dbProv.Save(ctx, &itemToSave)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	expectedRows := []map[string]interface{}{
		{
			"name":       itemToSave.Name,
			"created_at": *itemToSave.CreatedAt,
			"created_by": itemToSave.CreatedBy,
			"updated_at": *itemToSave.UpdatedAt,
			"updated_by": itemToSave.UpdatedBy,
			"cmd":        itemToSave.Cmd,
			"tags":       `["tag1","tag2"]`,
		},
	}
	q := "SELECT name, created_at, created_by, updated_at, updated_by, cmd, tags FROM `commands` WHERE id = ?"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{id})
}

func TestUpdate(t *testing.T) {
	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()
	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	itemToSave := demoData[0]
	itemToSave.Cmd = "awk"

	id, err := dbProv.Save(ctx, &itemToSave)
	require.NoError(t, err)
	assert.Equal(t, itemToSave.ID, id)
	expectedRows := []map[string]interface{}{
		{
			"id":          "1",
			"name":        itemToSave.Name,
			"created_at":  *itemToSave.CreatedAt,
			"created_by":  itemToSave.CreatedBy,
			"updated_at":  *itemToSave.UpdatedAt,
			"updated_by":  itemToSave.UpdatedBy,
			"cmd":         itemToSave.Cmd,
			"tags":        `["tag1","tag2"]`,
			"timeout_sec": int64(timeoutSec),
		},
	}
	q := "SELECT * FROM `commands` where id = ?"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{id})
}

func TestDelete(t *testing.T) {
	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	err = dbProv.Delete(ctx, "-2")
	assert.EqualError(t, err, "cannot find entry by id -2")

	err = dbProv.Delete(ctx, "2")
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"id":          "1",
			"name":        demoData[0].Name,
			"created_at":  *demoData[0].CreatedAt,
			"created_by":  demoData[0].CreatedBy,
			"updated_at":  *demoData[0].UpdatedAt,
			"updated_by":  demoData[0].UpdatedBy,
			"cmd":         demoData[0].Cmd,
			"tags":        `["tag1","tag2"]`,
			"timeout_sec": int64(timeoutSec),
		},
	}
	q := "SELECT * FROM `commands`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

func addDemoData(db *sqlx.DB) error {
	for i := range demoData {
		_, err := db.Exec(
			"INSERT INTO `commands` (`id`, `name`, `created_at`, `created_by`, `updated_at`, `updated_by`, `cmd`, `tags`) VALUES (?,?,?,?,?,?,?,?)",
			demoData[i].ID,
			demoData[i].Name,
			demoData[i].CreatedAt.Format(time.RFC3339),
			demoData[i].CreatedBy,
			demoData[i].UpdatedAt.Format(time.RFC3339),
			demoData[i].UpdatedBy,
			demoData[i].Cmd,
			demoData[i].Tags,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
