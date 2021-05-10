package vault

import (
	"context"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var testLog = chshare.NewLogger("client", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)

type configMock struct {
}

func (cm configMock) GetDatabasePath() string {
	return ":memory:"
}

func TestCallsBeforeInit(t *testing.T) {
	dbProv := NewSqliteProvider(configMock{}, testLog)

	expectedErrorText := "vault is not initialised yet"
	_, err := dbProv.GetStatus(context.Background())
	assert.EqualError(t, err, expectedErrorText)

	err = dbProv.SetStatus(context.Background(), DbStatus{})
	assert.EqualError(t, err, expectedErrorText)

	err = dbProv.Close()
	assert.NoError(t, err)
}

func TestSetStatus(t *testing.T) {
	dbProv := NewSqliteProvider(configMock{}, testLog)
	defer dbProv.Close()

	err := dbProv.Init(context.Background())
	require.NoError(t, err)

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
	dbProv := NewSqliteProvider(configMock{}, testLog)
	defer dbProv.Close()

	err := dbProv.Init(context.Background())
	require.NoError(t, err)

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
