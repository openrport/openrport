package clients

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseProvider(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("CREATE TABLE clients (id TEXT PRIMARY KEY, password TEXT)")
	require.NoError(t, err)
	c := &Client{ID: "test-client", Password: "test-password"}

	p := NewDatabaseProvider(db, "clients")

	// initial empty
	clients, err := p.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*Client{}, clients)

	// add new client
	added, err := p.Add(c)
	require.NoError(t, err)
	assert.True(t, added)

	// should contain client
	clients, err = p.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*Client{c}, clients)

	client, err := p.Get(c.ID)
	require.NoError(t, err)
	assert.Equal(t, c, client)

	// add existing client
	added, err = p.Add(c)
	require.NoError(t, err)
	assert.False(t, added)

	// delete client
	err = p.Delete(c.ID)
	require.NoError(t, err)

	// final empty
	clients, err = p.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*Client{}, clients)
}
