package vault

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/test"
)

var testLog = chshare.NewLogger("client", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)

type configMock struct {
}

func (cm configMock) GetDatabasePath() string {
	return ":memory:"
}

func TestSetStatus(t *testing.T) {
	dbProv, err := NewSqliteProvider(configMock{}, testLog)
	require.NoError(t, err)

	defer dbProv.Close()

	statusToSet := DbStatus{
		StatusName:    DbStatusInit,
		EncCheckValue: "123",
		DecCheckValue: "345",
	}
	err = dbProv.SetStatus(context.Background(), statusToSet)
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"db_status": DbStatusInit,
			"enc_check": "123",
			"dec_check": "345",
		},
	}
	query := "SELECT `db_status`, `enc_check`, `dec_check` FROM `status`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, query, []interface{}{})

	statusToSet.EncCheckValue = "678"
	statusToSet.DecCheckValue = "91011"
	statusToSet.StatusName = DbStatusNotInit

	err = dbProv.SetStatus(context.Background(), statusToSet)
	require.NoError(t, err)

	expectedRows2 := []map[string]interface{}{
		{
			"db_status": DbStatusNotInit,
			"enc_check": "678",
			"dec_check": "91011",
		},
	}
	test.AssertRowsEqual(t, dbProv.db, expectedRows2, query, []interface{}{})
}

func TestGetStatus(t *testing.T) {
	dbProv, err := NewSqliteProvider(configMock{}, testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	dbStatus, err := dbProv.GetStatus(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t,
		DbStatus{
			ID:            0,
			StatusName:    "",
			EncCheckValue: "",
			DecCheckValue: "",
		},
		dbStatus,
	)

	_, err = dbProv.db.Exec("INSERT INTO `status` (`db_status`, `enc_check`, `dec_check`) VALUES ('someStatus', 'someEnc', 'someDec')")
	require.NoError(t, err)

	dbStatus, err = dbProv.GetStatus(context.Background())
	require.NoError(t, err)

	assert.Equal(
		t,
		DbStatus{
			ID:            1,
			StatusName:    "someStatus",
			EncCheckValue: "someEnc",
			DecCheckValue: "someDec",
		},
		dbStatus,
	)
}

func TestGetByID(t *testing.T) {
	dbProv, err := NewSqliteProvider(configMock{}, testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	val, found, err := dbProv.GetByID(ctx, 1)

	require.NoError(t, err)
	require.True(t, found)
	expectedCreatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 00:00:00")
	require.NoError(t, err)
	assert.Equal(
		t,
		StoredValue{
			InputValue: InputValue{
				ClientID:      "client1",
				RequiredGroup: "group1",
				Key:           "key1",
				Value:         "val1",
				Type:          "type1",
			},
			ID:        1,
			CreatedAt: expectedCreatedAt,
			UpdatedAt: expectedCreatedAt,
			CreatedBy: "user1",
			UpdatedBy: nil,
		},
		val,
	)

	_, found, err = dbProv.GetByID(ctx, -2)
	require.NoError(t, err)
	require.False(t, found)
}

func TestList(t *testing.T) {
	dbProv, err := NewSqliteProvider(configMock{}, testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)
	expectedCreatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 00:00:00")
	require.NoError(t, err)

	vals, err := dbProv.List(context.Background(), &query.ListOptions{})
	require.NoError(t, err)
	assert.Equal(
		t,
		[]ValueKey{
			{
				ID:        1,
				ClientID:  "client1",
				CreatedBy: "user1",
				CreatedAt: expectedCreatedAt,
				Key:       "key1",
			},
			{
				ID:        2,
				ClientID:  "client2",
				CreatedBy: "user1",
				CreatedAt: expectedCreatedAt,
				Key:       "key2",
			},
		},
		vals,
	)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Sorts: []query.SortOption{
			{
				Column: "client_id",
				IsASC:  false,
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(
		t,
		[]ValueKey{
			{
				ID:        2,
				ClientID:  "client2",
				CreatedBy: "user1",
				CreatedAt: expectedCreatedAt,
				Key:       "key2",
			},
			{
				ID:        1,
				ClientID:  "client1",
				CreatedBy: "user1",
				CreatedAt: expectedCreatedAt,
				Key:       "key1",
			},
		},
		vals,
	)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Sorts: []query.SortOption{
			{
				Column: "key",
				IsASC:  true,
			},
		},
		Filters: []query.FilterOption{
			{
				Column: "created_by",
				Values: []string{"user1"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(
		t,
		[]ValueKey{
			{
				ID:        1,
				ClientID:  "client1",
				CreatedBy: "user1",
				CreatedAt: expectedCreatedAt,
				Key:       "key1",
			},
			{
				ID:        2,
				ClientID:  "client2",
				CreatedBy: "user1",
				CreatedAt: expectedCreatedAt,
				Key:       "key2",
			},
		},
		vals,
	)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Filters: []query.FilterOption{
			{
				Column: "client_id",
				Values: []string{"notExistingClient"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []ValueKey{}, vals)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Filters: []query.FilterOption{
			{
				Column: "key",
				Values: []string{"key1"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(
		t,
		[]ValueKey{
			{
				ID:        1,
				ClientID:  "client1",
				CreatedBy: "user1",
				CreatedAt: expectedCreatedAt,
				Key:       "key1",
			},
		},
		vals,
	)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Sorts: []query.SortOption{
			{
				Column: "key",
				IsASC:  true,
			},
		},
		Filters: []query.FilterOption{
			{
				Column: "key",
				Values: []string{"key1", "key2"},
			},
			{
				Column: "created_by",
				Values: []string{"user1"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(
		t,
		[]ValueKey{
			{
				ID:        1,
				ClientID:  "client1",
				CreatedBy: "user1",
				CreatedAt: expectedCreatedAt,
				Key:       "key1",
			},
			{
				ID:        2,
				ClientID:  "client2",
				CreatedBy: "user1",
				CreatedAt: expectedCreatedAt,
				Key:       "key2",
			},
		},
		vals,
	)
}

func TestCreate(t *testing.T) {
	dbProv, err := NewSqliteProvider(configMock{}, testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	expectedCreatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 00:00:00")
	require.NoError(t, err)

	ctx := context.Background()

	id, err := dbProv.Save(
		ctx,
		"user123",
		0,
		&InputValue{
			ClientID:      "client123",
			RequiredGroup: "group123",
			Key:           "key123",
			Value:         "value123",
			Type:          "typ123",
		},
		expectedCreatedAt,
	)
	require.NoError(t, err)
	assert.True(t, id > 0)

	expectedRows := []map[string]interface{}{
		{
			"id":             int64(1),
			"client_id":      "client123",
			"required_group": "group123",
			"created_at":     expectedCreatedAt,
			"created_by":     "user123",
			"updated_at":     expectedCreatedAt,
			"updated_by":     "user123",
			"key":            "key123",
			"value":          "value123",
			"type":           "typ123",
		},
	}
	query := "SELECT * FROM `values`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, query, []interface{}{})
}

func TestUpdate(t *testing.T) {
	dbProv, err := NewSqliteProvider(configMock{}, testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	expectedCreatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 00:00:00")
	require.NoError(t, err)

	expectedUpdatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-02 00:00:00")
	require.NoError(t, err)

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	id, err := dbProv.Save(
		ctx,
		"user123",
		1,
		&InputValue{
			ClientID:      "client123",
			RequiredGroup: "group123",
			Key:           "key123",
			Value:         "value123",
			Type:          "typ123",
		},
		expectedUpdatedAt,
	)
	require.NoError(t, err)
	assert.True(t, id > 0)

	expectedRows := []map[string]interface{}{
		{
			"id":             int64(1),
			"client_id":      "client123",
			"required_group": "group123",
			"created_at":     expectedCreatedAt,
			"created_by":     "user1",
			"updated_at":     expectedUpdatedAt,
			"updated_by":     "user123",
			"key":            "key123",
			"value":          "value123",
			"type":           "typ123",
		},
	}
	query := "SELECT * FROM `values` where id = 1"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, query, []interface{}{})
}

func TestFindByKeyAndClientID(t *testing.T) {
	dbProv, err := NewSqliteProvider(configMock{}, testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	expectedCreatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 00:00:00")
	require.NoError(t, err)

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	_, found, err := dbProv.FindByKeyAndClientID(ctx, "key1", "unknownClient")
	require.NoError(t, err)
	assert.False(t, found)

	_, found, err = dbProv.FindByKeyAndClientID(ctx, "unknownKey", "client1")
	require.NoError(t, err)
	assert.False(t, found)

	val, found, err := dbProv.FindByKeyAndClientID(ctx, "key1", "client1")
	require.NoError(t, err)
	assert.True(t, found)

	assert.Equal(
		t,
		StoredValue{
			InputValue: InputValue{
				ClientID:      "client1",
				RequiredGroup: "group1",
				Key:           "key1",
				Value:         "val1",
				Type:          "type1",
			},
			ID:        1,
			CreatedAt: expectedCreatedAt,
			UpdatedAt: expectedCreatedAt,
			CreatedBy: "user1",
			UpdatedBy: nil,
		},
		val,
	)
}

func TestDelete(t *testing.T) {
	dbProv, err := NewSqliteProvider(configMock{}, testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	ctx := context.Background()

	expectedCreatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 00:00:00")
	require.NoError(t, err)

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	err = dbProv.Delete(ctx, -2)
	assert.EqualError(t, err, "cannot find entry by id -2")

	err = dbProv.Delete(ctx, 1)
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"id":             int64(2),
			"client_id":      "client2",
			"required_group": "group1",
			"created_at":     expectedCreatedAt,
			"created_by":     "user1",
			"updated_at":     expectedCreatedAt,
			"updated_by":     nil,
			"key":            "key2",
			"value":          "val2",
			"type":           "type2",
		},
	}
	query := "SELECT * FROM `values`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, query, []interface{}{})
}

func addDemoData(db *sqlx.DB) error {
	demoDate, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 00:00:00")
	if err != nil {
		return err
	}
	demoData := []StoredValue{
		{
			InputValue: InputValue{
				ClientID:      "client1",
				RequiredGroup: "group1",
				Key:           "key1",
				Value:         "val1",
				Type:          "type1",
			},
			CreatedAt: demoDate,
			UpdatedAt: demoDate,
			CreatedBy: "user1",
			UpdatedBy: nil,
		},
		{
			InputValue: InputValue{
				ClientID:      "client2",
				RequiredGroup: "group1",
				Key:           "key2",
				Value:         "val2",
				Type:          "type2",
			},
			CreatedAt: demoDate,
			UpdatedAt: demoDate,
			CreatedBy: "user1",
			UpdatedBy: nil,
		},
	}

	for i := range demoData {
		_, err = db.Exec(
			"INSERT INTO `values` (`client_id`, `required_group`, `key`, `value`, `created_at`, `updated_at`, `created_by`, `type`) VALUES (?,?,?,?,?,?,?,?)",
			demoData[i].ClientID,
			demoData[i].RequiredGroup,
			demoData[i].Key,
			demoData[i].Value,
			demoData[i].CreatedAt.Format(time.RFC3339),
			demoData[i].UpdatedAt.Format(time.RFC3339),
			demoData[i].CreatedBy,
			demoData[i].Type,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
