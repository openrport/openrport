package authorization

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/db/migration/api_token"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/ptr"
	"github.com/cloudradar-monitoring/rport/share/test"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}
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
	{
		Username:  "username4",
		Prefix:    "prefix41",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "onescope41",
		Token:     "onelongtoken41",
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

func TestList(t *testing.T) {
	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	t.Cleanup(func() { dbProv.Close() })

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	result, err := dbProv.GetAll(context.Background(), "username1")
	require.NoError(t, err)
	assert.Equal(t, demoData[0], *result[0])

	result, err = dbProv.GetAll(context.Background(), "username4")
	require.NoError(t, err)
	assert.Equal(t, demoData[3], *result[0])
	assert.Equal(t, demoData[4], *result[1])
}

func TestCreate(t *testing.T) {

	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()

	require.NoError(t, err)

	ctx := context.Background()
	itemToSave := demoData[0]
	err = dbProv.Save(ctx, &itemToSave)
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
	err = dbProv.Save(ctx, &itemToSave)
	require.NoError(t, err)

	var demoDataUpdate = &APIToken{
		Username:  "username1",
		Prefix:    "prefix1",
		ExpiresAt: ptr.Time(time.Date(2011, 3, 11, 2, 0, 0, 0, time.UTC)),
	}

	err = dbProv.Save(ctx, demoDataUpdate)
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"username":   demoDataUpdate.Username,
			"prefix":     demoDataUpdate.Prefix,
			"expires_at": *demoDataUpdate.ExpiresAt,
		},
	}
	q := "SELECT username, prefix, expires_at FROM `api_token`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

func TestDelete(t *testing.T) {
	db, err := sqlite.New(":memory:", api_token.AssetNames(), api_token.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := NewSqliteProvider(db)
	defer dbProv.Close()

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	err = dbProv.Delete(ctx, "username1", "prefix2")
	assert.EqualError(t, err, "cannot find API Token by prefix prefix2")

	err = dbProv.Delete(ctx, "username2", "prefix2")
	require.NoError(t, err)

	err = dbProv.Delete(ctx, "username3", "prefix3")
	require.NoError(t, err)

	err = dbProv.Delete(ctx, "username4", "prefix4")
	require.NoError(t, err)

	err = dbProv.Delete(ctx, "username4", "prefix41")
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"username":   demoData[0].Username,
			"prefix":     demoData[0].Prefix,
			"created_at": *demoData[0].CreatedAt,
			"expires_at": *demoData[0].ExpiresAt,
			"scope":      demoData[0].Scope,
			"token":      demoData[0].Token,
		},
	}
	q := "SELECT * FROM `api_token`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

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
