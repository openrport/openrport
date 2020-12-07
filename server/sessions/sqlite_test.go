package sessions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientSessionsSqliteProvider(t *testing.T) {
	ctx := context.Background()
	keepLost := hour
	p := newFakeSessionProvider(t, keepLost)
	defer p.Close()

	// verify add sessions
	s1 := New(t).Build()                                                   // active
	s2 := New(t).DisconnectedDuration(5 * time.Minute).Build()             // disconnected
	s3 := New(t).DisconnectedDuration(keepLost - time.Millisecond).Build() // disconnected
	s4 := New(t).DisconnectedDuration(keepLost).Build()                    // disconnected
	s5 := New(t).DisconnectedDuration(keepLost + time.Millisecond).Build() // obsolete
	require.NoError(t, p.Save(ctx, s1))
	require.NoError(t, p.Save(ctx, s2))
	require.NoError(t, p.Save(ctx, s3))
	require.NoError(t, p.Save(ctx, s4))
	require.NoError(t, p.Save(ctx, s5))

	// verify get sessions
	gotAll, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*ClientSession{s1, s2, s3, s4}, gotAll)

	// verify delete obsolete sessions
	gotObsolete, err := p.get(ctx, s5.ID)
	require.NoError(t, err)
	require.EqualValues(t, s5, gotObsolete)

	require.NoError(t, p.DeleteObsolete(ctx))
	gotObsolete, err = p.get(ctx, s5.ID)
	require.NoError(t, err)
	require.Nil(t, gotObsolete)

	gotAll, err = p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*ClientSession{s1, s2, s3, s4}, gotAll)

	// verify not found
	gotNone, err := p.get(ctx, "unknown-id")
	require.NoError(t, err)
	require.Nil(t, gotNone)

	// verify update
	d := time.Date(2020, 11, 5, 12, 11, 20, 0, time.UTC)
	s1.Disconnected = &d
	require.NoError(t, p.Save(ctx, s1))
	gotUpdated, err := p.get(ctx, s1.ID)
	require.NoError(t, err)
	require.EqualValues(t, s1, gotUpdated)
	gotAll, err = p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*ClientSession{s1, s2, s3, s4}, gotAll)
}
