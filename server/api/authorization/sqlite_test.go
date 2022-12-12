package authorization

import (
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/db/migration/api_token"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/ptr"
	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/require"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}
var timeoutSec = DefaultTimeoutSec
var demoData = []APIToken{
	{
		Username:  "user1",
		Prefix:    "dshjdsfhj",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "dshjdsfhj",
		Token:     "ddsfsddfsfdsfsdsfdsdfshjdsfhj",
	},
}

func TestGetByID(t *testing.T) {
	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()

	// ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	// val, found, err := dbProv.GetByID(ctx, "1", &query.RetrieveOptions{})

	// require.NoError(t, err)
	// require.True(t, found)
	// require.NoError(t, err)
	// assert.Equal(t, demoData[0], *val)

	// _, found, err = dbProv.GetByID(ctx, "-2", &query.RetrieveOptions{})
	// require.NoError(t, err)
	// require.False(t, found)

	// val, found, err = dbProv.GetByID(ctx, "1", &query.RetrieveOptions{Fields: []query.FieldsOption{
	// 	{
	// 		Resource: "scripts",
	// 		Fields:   []string{"created_by", "script"},
	// 	},
	// }})

	// require.NoError(t, err)
	// require.True(t, found)
	// require.NoError(t, err)
	// assert.Equal(t, Script{
	// 	CreatedBy: "user1",
	// 	Script:    "ls -la",
	// }, *val)
}

// func TestList(t *testing.T) {
// 	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
// 	require.NoError(t, err)
// 	dbProv := NewSqliteProvider(db)
// 	t.Cleanup(func() { dbProv.Close() })

// 	err = addDemoData(dbProv.db)
// 	require.NoError(t, err)

// 	testCases := []struct {
// 		Name           string
// 		Options        *query.ListOptions
// 		ExpectedResult []Script
// 	}{
// 		{
// 			Name:           "no options",
// 			Options:        &query.ListOptions{},
// 			ExpectedResult: demoData,
// 		},
// 		{
// 			Name: "sort only",
// 			Options: &query.ListOptions{
// 				Sorts: []query.SortOption{
// 					{
// 						Column: "created_at",
// 						IsASC:  false,
// 					},
// 				},
// 			},
// 			ExpectedResult: []Script{demoData[1], demoData[0]},
// 		},
// 		{
// 			Name: "filter and sort",
// 			Options: &query.ListOptions{
// 				Sorts: []query.SortOption{
// 					{
// 						Column: "name",
// 						IsASC:  true,
// 					},
// 				},
// 				Filters: []query.FilterOption{
// 					{
// 						Column: []string{"created_by"},
// 						Values: []string{"user1"},
// 					},
// 				},
// 			},
// 			ExpectedResult: []Script{demoData[1], demoData[0]},
// 		},
// 		{
// 			Name: "filter, no results",
// 			Options: &query.ListOptions{
// 				Filters: []query.FilterOption{
// 					{
// 						Column: []string{"interpreter"},
// 						Values: []string{"not-existing-interpreter"},
// 					},
// 				},
// 			},
// 			ExpectedResult: []Script{},
// 		},
// 		{
// 			Name: "filter, 1 result",
// 			Options: &query.ListOptions{
// 				Filters: []query.FilterOption{
// 					{
// 						Column: []string{"is_sudo"},
// 						Values: []string{"0"},
// 					},
// 				},
// 			},
// 			ExpectedResult: []Script{demoData[0]},
// 		},
// 		{
// 			Name: "multiple filters",
// 			Options: &query.ListOptions{
// 				Sorts: []query.SortOption{
// 					{
// 						Column: "interpreter",
// 						IsASC:  true,
// 					},
// 				},
// 				Filters: []query.FilterOption{
// 					{
// 						Column: []string{"name"},
// 						Values: []string{"some name", "other name 2"},
// 					},
// 					{
// 						Column: []string{"created_by"},
// 						Values: []string{"user1"},
// 					},
// 				},
// 			},
// 			ExpectedResult: demoData,
// 		},
// 		{
// 			Name: "fields",
// 			Options: &query.ListOptions{
// 				Fields: []query.FieldsOption{
// 					{
// 						Resource: "scripts",
// 						Fields:   []string{"id", "name"},
// 					},
// 				},
// 			},
// 			ExpectedResult: []Script{
// 				{
// 					ID:   "1",
// 					Name: "some name",
// 				},
// 				{
// 					ID:   "2",
// 					Name: "other name 2",
// 				},
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc
// 		t.Run(tc.Name, func(t *testing.T) {
// 			result, err := dbProv.List(context.Background(), tc.Options)
// 			require.NoError(t, err)

// 			assert.Equal(t, tc.ExpectedResult, result)
// 		})
// 	}
// }

// func TestCreate(t *testing.T) {
// 	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
// 	require.NoError(t, err)
// 	dbProv := NewSqliteProvider(db)
// 	defer dbProv.Close()

// 	expectedCreatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 01:00:00")
// 	require.NoError(t, err)

// 	ctx := context.Background()
// 	itemToSave := demoData[0]
// 	itemToSave.ID = ""
// 	id, err := dbProv.Save(ctx, &itemToSave, expectedCreatedAt.UTC())
// 	require.NoError(t, err)
// 	assert.True(t, id != "")

// 	expectedRows := []map[string]interface{}{
// 		{
// 			"name":        itemToSave.Name,
// 			"created_at":  *itemToSave.CreatedAt,
// 			"created_by":  itemToSave.CreatedBy,
// 			"updated_at":  *itemToSave.UpdatedAt,
// 			"updated_by":  itemToSave.UpdatedBy,
// 			"interpreter": *itemToSave.Interpreter,
// 			"is_sudo":     int64(0),
// 			"cwd":         *itemToSave.Cwd,
// 			"script":      itemToSave.Script,
// 			"tags":        `["tag1","tag2"]`,
// 		},
// 	}
// 	q := "SELECT name, created_at, created_by, updated_at, updated_by, interpreter, is_sudo, cwd, script, tags FROM `scripts`"
// 	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
// }

// func TestUpdate(t *testing.T) {
// 	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
// 	require.NoError(t, err)
// 	dbProv := NewSqliteProvider(db)
// 	defer dbProv.Close()

// 	ctx := context.Background()

// 	err = addDemoData(dbProv.db)
// 	require.NoError(t, err)

// 	itemToSave := demoData[0]
// 	itemToSave.Script = "awk"

// 	id, err := dbProv.Save(
// 		ctx,
// 		&itemToSave,
// 		time.Date(2012, 1, 1, 1, 0, 0, 0, time.UTC),
// 	)
// 	require.NoError(t, err)
// 	assert.Equal(t, itemToSave.ID, id)

// 	expectedRows := []map[string]interface{}{
// 		{
// 			"id":          "1",
// 			"name":        itemToSave.Name,
// 			"created_at":  *itemToSave.CreatedAt,
// 			"created_by":  itemToSave.CreatedBy,
// 			"updated_at":  *itemToSave.UpdatedAt,
// 			"updated_by":  itemToSave.UpdatedBy,
// 			"interpreter": *itemToSave.Interpreter,
// 			"is_sudo":     int64(0),
// 			"cwd":         *itemToSave.Cwd,
// 			"script":      itemToSave.Script,
// 			"tags":        `["tag1","tag2"]`,
// 			"timeout_sec": int64(timeoutSec),
// 		},
// 	}
// 	q := "SELECT * FROM `scripts` where id = 1"
// 	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
// }

// func TestDelete(t *testing.T) {
// 	db, err := sqlite.New(":memory:", library.AssetNames(), library.Asset, DataSourceOptions)
// 	require.NoError(t, err)
// 	dbProv := NewSqliteProvider(db)
// 	defer dbProv.Close()

// 	ctx := context.Background()

// 	err = addDemoData(dbProv.db)
// 	require.NoError(t, err)

// 	err = dbProv.Delete(ctx, "-2")
// 	assert.EqualError(t, err, "cannot find entry by id -2")

// 	err = dbProv.Delete(ctx, "2")
// 	require.NoError(t, err)

// 	expectedRows := []map[string]interface{}{
// 		{
// 			"id":          "1",
// 			"name":        demoData[0].Name,
// 			"created_at":  *demoData[0].CreatedAt,
// 			"created_by":  demoData[0].CreatedBy,
// 			"updated_at":  *demoData[0].UpdatedAt,
// 			"updated_by":  demoData[0].UpdatedBy,
// 			"interpreter": *demoData[0].Interpreter,
// 			"is_sudo":     int64(0),
// 			"cwd":         *demoData[0].Cwd,
// 			"script":      demoData[0].Script,
// 			"tags":        `["tag1","tag2"]`,
// 			"timeout_sec": int64(timeoutSec),
// 		},
// 	}
// 	q := "SELECT * FROM `scripts`"
// 	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
// }

func addDemoData(db *sqlx.DB) error {
	for i := range demoData {
		_, err := db.Exec(
			"INSERT INTO `api_token` (`username`, `prefix`, `created_at`, `expires_at`, `scope`, `token`) VALUES (?,?,?,?,?,?)",
			demoData[i].Username,
			demoData[i].Prefix,
			demoData[i].CreatedAt.Format(time.RFC3339),
			demoData[i].ExpiresAt.Format(time.RFC3339),
			demoData[i].Scope,
			demoData[i].Token,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
