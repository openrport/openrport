package auditlog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/db/migration/auditlog"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/test"
)

func TestSqliteSave(t *testing.T) {
	db, err := sqlite.New(":memory:", auditlog.AssetNames(), auditlog.Asset)
	require.NoError(t, err)
	dbProv := SQLiteProvider{
		db: db,
	}
	defer dbProv.Close()

	e := &Entry{
		Timestamp:      time.Date(2021, 10, 19, 13, 57, 58, 0, time.UTC),
		Username:       "admin",
		RemoteIP:       "192.168.55.23",
		Application:    ApplicationLibraryCommand,
		Action:         ActionCreate,
		ID:             "db4960c8-c7dd-42b8-9db6-2a98dc7122d9",
		ClientID:       "e9e7e70c-d023-4423-869c-86d70da8f243",
		ClientHostName: "127.0.0.1",
		Request:        `{"k1": "v1"}`,
		Response:       `{"k1": "v1"}`,
	}
	err = dbProv.Save(e)
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"timestamp":       e.Timestamp,
			"username":        e.Username,
			"remote_ip":       e.RemoteIP,
			"application":     e.Application,
			"action":          e.Action,
			"affected_id":     e.ID,
			"client_id":       e.ClientID,
			"client_hostname": e.ClientHostName,
			"request":         e.Request,
			"response":        e.Response,
		},
	}
	q := "SELECT * FROM auditlog"
	test.AssertRowsEqual(t, db, expectedRows, q, []interface{}{})
}
