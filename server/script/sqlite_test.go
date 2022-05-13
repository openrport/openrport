package script

import (
	"context"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/db/migration/library"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/ptr"
	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/share/test"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}

var demoData = []Script{
	{
		ID:          "1",
		Name:        "some name",
		CreatedBy:   "user1",
		CreatedAt:   ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		Interpreter: ptr.String("bash"),
		IsSudo:      ptr.Bool(false),
		Cwd:         ptr.String("/bin"),
		Script:      "ls -la",
	},
	{
		ID:          "2",
		Name:        "other name 2",
		CreatedBy:   "user1",
		CreatedAt:   ptr.Time(time.Date(2002, 1, 1, 1, 0, 0, 0, time.UTC)),
		Interpreter: ptr.String("sh"),
		IsSudo:      ptr.Bool(true),
		Cwd:         ptr.String("/bin"),
		Script:      "pwd",
	},
}

func TestGetByID(t *testing.T) {
	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	val, found, err := dbProv.GetByID(ctx, "1", &query.RetrieveOptions{})

	require.NoError(t, err)
	require.True(t, found)
	require.NoError(t, err)
	assert.Equal(t, demoData[0], *val)

	_, found, err = dbProv.GetByID(ctx, "-2", &query.RetrieveOptions{})
	require.NoError(t, err)
	require.False(t, found)

	val, found, err = dbProv.GetByID(ctx, "1", &query.RetrieveOptions{Fields: []query.FieldsOption{
		{
			Resource: "scripts",
			Fields:   []string{"created_by", "script"},
		},
	}})

	require.NoError(t, err)
	require.True(t, found)
	require.NoError(t, err)
	assert.Equal(t, Script{
		CreatedBy: "user1",
		Script:    "ls -la",
	}, *val)
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
		ExpectedResult []Script
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
			ExpectedResult: []Script{demoData[1], demoData[0]},
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
			ExpectedResult: []Script{demoData[1], demoData[0]},
		},
		{
			Name: "filter, no results",
			Options: &query.ListOptions{
				Filters: []query.FilterOption{
					{
						Column: []string{"interpreter"},
						Values: []string{"not-existing-interpreter"},
					},
				},
			},
			ExpectedResult: []Script{},
		},
		{
			Name: "filter, 1 result",
			Options: &query.ListOptions{
				Filters: []query.FilterOption{
					{
						Column: []string{"is_sudo"},
						Values: []string{"0"},
					},
				},
			},
			ExpectedResult: []Script{demoData[0]},
		},
		{
			Name: "multiple filters",
			Options: &query.ListOptions{
				Sorts: []query.SortOption{
					{
						Column: "interpreter",
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
						Resource: "scripts",
						Fields:   []string{"id", "name"},
					},
				},
			},
			ExpectedResult: []Script{
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

	expectedCreatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 01:00:00")
	require.NoError(t, err)

	ctx := context.Background()
	itemToSave := demoData[0]
	itemToSave.ID = ""
	id, err := dbProv.Save(ctx, &itemToSave, expectedCreatedAt.UTC())
	require.NoError(t, err)
	assert.True(t, id != "")

	expectedRows := []map[string]interface{}{
		{
			"name":        itemToSave.Name,
			"created_at":  *itemToSave.CreatedAt,
			"created_by":  itemToSave.CreatedBy,
			"interpreter": *itemToSave.Interpreter,
			"is_sudo":     int64(0),
			"cwd":         *itemToSave.Cwd,
			"script":      itemToSave.Script,
		},
	}
	q := "SELECT name, created_at, created_by, interpreter, is_sudo, cwd, script FROM `scripts`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
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
	itemToSave.Script = "awk"

	id, err := dbProv.Save(
		ctx,
		&itemToSave,
		time.Date(2012, 1, 1, 1, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	assert.Equal(t, itemToSave.ID, id)

	expectedRows := []map[string]interface{}{
		{
			"id":          "1",
			"name":        itemToSave.Name,
			"created_at":  *itemToSave.CreatedAt,
			"created_by":  itemToSave.CreatedBy,
			"interpreter": *itemToSave.Interpreter,
			"is_sudo":     int64(0),
			"cwd":         *itemToSave.Cwd,
			"script":      itemToSave.Script,
		},
	}
	q := "SELECT * FROM `scripts` where id = 1"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
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
			"interpreter": *demoData[0].Interpreter,
			"is_sudo":     int64(0),
			"cwd":         *demoData[0].Cwd,
			"script":      demoData[0].Script,
		},
	}
	q := "SELECT * FROM `scripts`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

func addDemoData(db *sqlx.DB) error {
	for i := range demoData {
		_, err := db.Exec(
			"INSERT INTO `scripts` (`id`, `name`, `created_at`, `created_by`, `interpreter`, `is_sudo`, `cwd`, `script`) VALUES (?,?,?,?,?,?,?,?)",
			demoData[i].ID,
			demoData[i].Name,
			demoData[i].CreatedAt.Format(time.RFC3339),
			demoData[i].CreatedBy,
			demoData[i].Interpreter,
			demoData[i].IsSudo,
			demoData[i].Cwd,
			demoData[i].Script,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
