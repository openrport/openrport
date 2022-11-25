package clients

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanup(t *testing.T) {
	// given
	ctx := context.Background()
	c1 := New(t).Build()                                               // active
	c2 := New(t).DisconnectedDuration(5 * time.Minute).Build()         // disconnected
	c3 := New(t).DisconnectedDuration(time.Hour + time.Minute).Build() // obsolete
	clients := []*Client{c1, c2, c3}
	p := NewFakeClientProvider(t, &hour, c1, c2, c3)
	defer p.Close()
	clientsRepo := NewClientRepositoryWithDB(clients, &hour, p, testLog)
	require.Len(t, clientsRepo.clients, 3)
	gotObsolete, err := p.get(ctx, c3.ID)
	require.NoError(t, err)
	require.EqualValues(t, c3, gotObsolete)
	task := NewCleanupTask(testLog, clientsRepo)

	// when
	err = task.Run(ctx)

	// then
	assert.NoError(t, err)
	assert.ElementsMatch(t, getValues(clientsRepo.clients), []*Client{c1, c2})
	gotClients, err := p.GetAll(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []*Client{c1, c2}, gotClients)
	gotObsolete, err = p.get(ctx, c3.ID)
	require.NoError(t, err)
	require.Nil(t, gotObsolete)
}

func TestCleanupDisabled(t *testing.T) {
	// given
	ctx := context.Background()
	c1 := New(t).Build()                                                      // active
	c2 := New(t).DisconnectedDuration(5 * time.Minute).Build()                // disconnected
	c3 := New(t).DisconnectedDuration(365*24*time.Hour + time.Minute).Build() // disconnected longer
	clients := []*Client{c1, c2, c3}
	p := NewFakeClientProvider(t, nil, c1, c2, c3)
	defer p.Close()
	clientsRepo := NewClientRepositoryWithDB(clients, nil, p, testLog)
	require.Len(t, clientsRepo.clients, 3)

	task := NewCleanupTask(testLog, clientsRepo)

	// when
	err := task.Run(ctx)

	// then
	assert.NoError(t, err)
	assert.ElementsMatch(t, getValues(clientsRepo.clients), []*Client{c1, c2, c3})
}

func getValues(clients map[string]*Client) []*Client {
	var r []*Client
	for _, v := range clients {
		r = append(r, v)
	}
	return r
}
