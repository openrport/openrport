package authorization

import (
	"context"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/db/migration/api_token"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/ptr"
	"github.com/cloudradar-monitoring/rport/share/test"
	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}
var timeoutSec = DefaultTimeoutSec
var demoData = []APIToken{
	{
		Username:  "username1",
		Prefix:    "prefix1",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "onescope1",
		Token:     "onelongtoken1",
	},
	{
		Username:  "username2",
		Prefix:    "prefix2",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "onescope2",
		Token:     "onelongtoken2",
	},
	{
		Username:  "username3",
		Prefix:    "prefix3",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "onescope3",
		Token:     "onelongtoken3",
	},
	{
		Username:  "username4",
		Prefix:    "prefix4",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "onescope4",
		Token:     "onelongtoken4",
	},
}

func TestGet(t *testing.T) {
	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	val, err := dbProv.Get(ctx, "username1", "prefix1")

	require.NoError(t, err)
	require.NotNil(t, val)
	assert.Equal(t, demoData[0], *val)

	val, err = dbProv.Get(ctx, "username1", "prefix3")

	require.NoError(t, err)
	require.Nil(t, val)

}

// func TestList(t *testing.T) {
// 	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
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

func TestCreate(t *testing.T) {

	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()

	require.NoError(t, err)

	ctx := context.Background()
	itemToSave := demoData[0]
	err = dbProv.save(ctx, &itemToSave)
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"username":   itemToSave.Username,
			"prefix":     itemToSave.Prefix,
			"expires_at": *itemToSave.ExpiresAt,
			"scope":      itemToSave.Scope,
			"token":      itemToSave.Token,
		},
	}
	q := "SELECT username, prefix, expires_at, scope, token FROM `api_token`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

func TestUpdate(t *testing.T) {

	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()

	require.NoError(t, err)

	ctx := context.Background()
	itemToSave := demoData[0]
	err = dbProv.save(ctx, &itemToSave)
	require.NoError(t, err)

	var demoDataUpdate = &APIToken{
		Username:  "username1",
		Prefix:    "prefix1",
		ExpiresAt: ptr.Time(time.Date(2011, 3, 11, 2, 0, 0, 0, time.UTC)),
		Scope:     "onenewscope1",
		Token:     "onenewlongtoken1",
	}

	err = dbProv.save(ctx, demoDataUpdate)
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"username":   demoDataUpdate.Username,
			"prefix":     demoDataUpdate.Prefix,
			"expires_at": *demoDataUpdate.ExpiresAt,
			"scope":      demoDataUpdate.Scope,
			"token":      demoDataUpdate.Token,
		},
	}
	q := "SELECT username, prefix, expires_at, scope, token FROM `api_token`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

// func TestUpdate(t *testing.T) {
// 	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
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
// 	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
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
