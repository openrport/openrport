package authorization

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/db/migration/api_token"
	"github.com/realvnc-labs/rport/db/sqlite"
	"github.com/realvnc-labs/rport/share/ptr"
	"github.com/realvnc-labs/rport/share/test"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}
var demoData = []APIToken{
	{
		Username:  "username1",
		Prefix:    "prefix1",
		Name:      "This is a token name 1",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "read",
		Token:     "onelongtoken1",
	},
	{
		Username:  "username2",
		Prefix:    "prefix2",
		Name:      "This is a token name 2",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "read",
		Token:     "onelongtoken2",
	},
	{
		Username:  "username3",
		Prefix:    "prefix3",
		Name:      "This is a token name 3",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "read+write",
		Token:     "onelongtoken3",
	},
	{
		Username:  "username4",
		Prefix:    "prefix4",
		Name:      "This is a token name 4",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "read+write",
		Token:     "onelongtoken4",
	},
	{
		Username:  "username4",
		Prefix:    "prefix41",
		Name:      "This is a token name 41",
		CreatedAt: ptr.Time(time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)),
		ExpiresAt: ptr.Time(time.Date(2001, 1, 1, 2, 0, 0, 0, time.UTC)),
		Scope:     "read+write",
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

	result, err := dbProv.GetAllForUser(context.Background(), "username1")
	require.NoError(t, err)
	assert.Equal(t, demoData[0], *result[0])

	result, err = dbProv.GetAllForUser(context.Background(), "username4")
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
			"name":       itemToSave.Name,
			"expires_at": *itemToSave.ExpiresAt,
			"scope":      "read", // needed to avoid test fail using itemToSave.Scope which is of type enum
			"token":      itemToSave.Token,
		},
	}
	q := "SELECT username, prefix, name, expires_at, scope, token FROM `api_tokens`"

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
		Name:      "A token name",
		ExpiresAt: ptr.Time(time.Date(2011, 3, 11, 2, 0, 0, 0, time.UTC)),
	}

	err = dbProv.Save(ctx, demoDataUpdate)
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"username":   demoDataUpdate.Username,
			"prefix":     demoDataUpdate.Prefix,
			"name":       demoDataUpdate.Name,
			"expires_at": *demoDataUpdate.ExpiresAt,
		},
	}
	q := "SELECT username, prefix, name, expires_at FROM `api_tokens`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}
func TestUpdateOneAtATimeName(t *testing.T) {
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
		Username: "username1",
		Prefix:   "prefix1",
		Name:     "a brand new name",
	}

	err = dbProv.Save(ctx, demoDataUpdate)
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"username":   demoDataUpdate.Username,
			"prefix":     demoDataUpdate.Prefix,
			"name":       demoDataUpdate.Name,
			"expires_at": *itemToSave.ExpiresAt,
		},
	}
	q := "SELECT username, prefix, name, expires_at FROM `api_tokens`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

func TestUpdateOneAtATimeExpiresAt(t *testing.T) {
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
			"name":       itemToSave.Name,
			"expires_at": *demoDataUpdate.ExpiresAt,
		},
	}
	q := "SELECT username, prefix, name, expires_at FROM `api_tokens`"
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
			"name":       demoData[0].Name,
			"created_at": *demoData[0].CreatedAt,
			"expires_at": *demoData[0].ExpiresAt,
			"scope":      "read", // needed to avoid test fail using itemToSave.Scope which is of type enum
			"token":      demoData[0].Token,
		},
	}
	q := "SELECT * FROM `api_tokens`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

func addDemoData(db *sqlx.DB) error {
	for i := range demoData {
		_, err := db.Exec(
			"INSERT INTO `api_tokens` (`username`, `prefix`, `name`, `created_at`, `expires_at`, `scope`, `token`) VALUES (?,?,?,?,?,?,?)",
			demoData[i].Username,
			demoData[i].Prefix,
			demoData[i].Name,
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
