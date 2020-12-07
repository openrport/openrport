package sessions

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
	s1 := New(t).Build()                                               // active
	s2 := New(t).DisconnectedDuration(5 * time.Minute).Build()         // disconnected
	s3 := New(t).DisconnectedDuration(time.Hour + time.Minute).Build() // obsolete
	sessions := []*ClientSession{s1, s2, s3}
	repo := NewSessionRepository(sessions, &hour)
	require.Len(t, repo.sessions, 3)
	p := newFakeSessionProvider(t, hour, s1, s2, s3)
	defer p.Close()
	gotObsolete, err := p.get(ctx, s3.ID)
	require.NoError(t, err)
	require.EqualValues(t, s3, gotObsolete)
	task := NewCleanupTask(testLog, repo, p)

	// when
	err = task.Run(ctx)

	// then
	assert.NoError(t, err)
	assert.ElementsMatch(t, getValues(repo.sessions), []*ClientSession{s1, s2})
	gotSessions, err := p.GetAll(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, []*ClientSession{s1, s2}, gotSessions)
	gotObsolete, err = p.get(ctx, s3.ID)
	require.NoError(t, err)
	require.Nil(t, gotObsolete)
}

func getValues(sessions map[string]*ClientSession) []*ClientSession {
	var r []*ClientSession
	for _, v := range sessions {
		r = append(r, v)
	}
	return r
}
