package clients

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveTask(t *testing.T) {
	// given
	ctx := context.Background()
	exp := 2 * time.Hour
	// keepLostClients 2 hours: 1 active (c1), 2 disconnected(c2, c3, c4), 1 obsolete(c5)
	c1 := New(t).Build()
	c2 := New(t).DisconnectedDuration(5 * time.Minute).Build()
	c3 := New(t).DisconnectedDuration(time.Hour).Build()
	c4 := New(t).DisconnectedDuration(time.Hour + time.Minute).Build()
	c5 := New(t).DisconnectedDuration(3 * time.Hour).Build()
	clients := []*Client{c1, c2, c3, c4, c5}
	p := newFakeClientProvider(t, exp)
	defer p.Close()
	task := NewSaveTask(testLog, NewClientRepository(clients, &exp), p)

	// when
	err := task.Run(ctx)

	// then
	assert.NoError(t, err)
	gotAll, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*Client{c1, c2, c3, c4}, gotAll)

	// keepLostClients 1 hour: 1 active (c1), 1 disconnected(c2, c3), 2 obsolete(c4, c5)
	p.keepLostClients = hour
	gotClients, err := GetInitState(ctx, p)
	assert.NoError(t, err)
	wantC1 := shallowCopy(c1)
	wantC1.DisconnectedAt = &nowMock
	require.Len(t, gotClients, 3)
	assert.ElementsMatch(t, gotClients, []*Client{wantC1, c2, c3})
}
