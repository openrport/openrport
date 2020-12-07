package sessions

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
	// keepLostClients 2 hours: 1 active (s1), 2 disconnected(s2, s3, s4), 1 obsolete(s5)
	s1 := New(t).Build()
	s2 := New(t).DisconnectedDuration(5 * time.Minute).Build()
	s3 := New(t).DisconnectedDuration(time.Hour).Build()
	s4 := New(t).DisconnectedDuration(time.Hour + time.Minute).Build()
	s5 := New(t).DisconnectedDuration(3 * time.Hour).Build()
	sessions := []*ClientSession{s1, s2, s3, s4, s5}
	p := newFakeSessionProvider(t, exp)
	defer p.Close()
	task := NewSaveTask(testLog, NewSessionRepository(sessions, &exp), p)

	// when
	err := task.Run(ctx)

	// then
	assert.NoError(t, err)
	gotAll, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*ClientSession{s1, s2, s3, s4}, gotAll)

	// keepLostClients 1 hour: 1 active (s1), 1 disconnected(s2, s3), 2 obsolete(s4, s5)
	p.keepLostClients = hour
	gotSessions, err := GetInitState(ctx, p)
	assert.NoError(t, err)
	wantS1 := shallowCopy(s1)
	wantS1.Disconnected = &nowMock
	require.Len(t, gotSessions, 3)
	assert.ElementsMatch(t, gotSessions, []*ClientSession{wantS1, s2, s3})
}
