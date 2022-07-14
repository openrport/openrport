package clientsauth

import (
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/jmoiron/sqlx"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

// NewDatabaseMockProvider creates a clients auth database for testing and returns the DatabaseProvider
func NewDatabaseMockProvider(clients []*ClientAuth, t *testing.T) *DatabaseProvider {
	var authDb *sqlx.DB
	authDb, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		require.NoError(t, err)
	}
	if _, err := authDb.Exec(`CREATE TABLE clients_auth (id text,password text)`); err != nil {
		log.Fatalln(err)
	}
	for _, v := range clients {
		if _, err := authDb.Exec("INSERT INTO clients_auth VALUES(?,?)", v.ID, v.Password); err != nil {
			require.NoError(t, err)
		}
	}
	return &DatabaseProvider{
		db:        authDb,
		tableName: "clients_auth",
		converter: query.NewSQLConverter(authDb.DriverName()),
	}
}

// NewMockFileProvider creates a clients auth file for testing and returns the FileProvider
func NewMockFileProvider(clients []*ClientAuth, t *testing.T) *FileProvider {
	var authFile = t.TempDir() + "/client-auth.json"
	f, _ := os.Create(authFile)
	defer f.Close()
	clientAuth := make(map[string]string)
	for _, v := range clients {
		clientAuth[v.ID] = v.Password
	}
	cj, _ := json.Marshal(clientAuth)
	if _, err := f.Write(cj); err != nil {
		require.NoError(t, err)
	}
	return &FileProvider{
		fileName: authFile,
		cache:    cache.New(60*time.Minute, 15*time.Minute),
	}
}
