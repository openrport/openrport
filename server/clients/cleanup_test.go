package clients

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openrport/openrport/server/clients/clientdata"
)

func TestCleanup(t *testing.T) {
	// given
	ctx := context.Background()
	c1 := New(t).ID("client-1").Logger(testLog).Build()                                               // active
	c2 := New(t).ID("client-2").DisconnectedDuration(5 * time.Minute).Logger(testLog).Build()         // disconnected
	c3 := New(t).ID("client-3").DisconnectedDuration(time.Hour + time.Minute).Logger(testLog).Build() // obsolete
	clients := []*clientdata.Client{c1, c2, c3}
	p := NewFakeClientProvider(t, &hour, c1, c2, c3)
	defer p.Close()
	clientsRepo := NewClientRepositoryWithDB(clients, &hour, p, testLog)
	require.Len(t, clientsRepo.clientState, 3)
	gotObsolete, err := p.get(ctx, c3.GetID(), testLog)
	require.NoError(t, err)

	require.EqualValues(t, c3, gotObsolete)
	task := NewCleanupTask(testLog, clientsRepo)

	// when
	err = task.Run(ctx)

	// then
	assert.NoError(t, err)
	assert.ElementsMatch(t, getValues(clientsRepo.clientState), []*clientdata.Client{c1, c2})
	gotClients, err := p.GetAll(ctx, testLog)
	assert.NoError(t, err)

	assert.ElementsMatch(t, []*clientdata.Client{c1, c2}, gotClients)
	gotObsolete, err = p.get(ctx, c3.GetID(), testLog)
	require.NoError(t, err)
	require.Nil(t, gotObsolete)
}

func TestCleanupDisabled(t *testing.T) {
	// given
	ctx := context.Background()
	c1 := New(t).Logger(testLog).Build()                                                      // active
	c2 := New(t).DisconnectedDuration(5 * time.Minute).Logger(testLog).Build()                // disconnected
	c3 := New(t).DisconnectedDuration(365*24*time.Hour + time.Minute).Logger(testLog).Build() // disconnected longer
	clients := []*clientdata.Client{c1, c2, c3}
	p := NewFakeClientProvider(t, nil, c1, c2, c3)
	defer p.Close()
	clientsRepo := NewClientRepositoryWithDB(clients, nil, p, testLog)
	require.Len(t, clientsRepo.clientState, 3)

	task := NewCleanupTask(testLog, clientsRepo)

	// when
	err := task.Run(ctx)

	// then
	assert.NoError(t, err)
	assert.ElementsMatch(t, getValues(clientsRepo.clientState), []*clientdata.Client{c1, c2, c3})
}

func getValues(clients map[string]*clientdata.Client) []*clientdata.Client {
	var r []*clientdata.Client
	for _, v := range clients {
		r = append(r, v)
	}
	return r
}
