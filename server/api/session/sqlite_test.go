package session

import (
	"context"
	"testing"
	"time"

	"github.com/openrport/openrport/db/sqlite"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}

func TestAPISessionSqlite(t *testing.T) {
	ctx := context.Background()

	longTTL := time.Hour
	negativeTTL := -time.Minute

	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user1", timeNow.Add(longTTL), timeNow)
	s3 := generateAPISession(t, "user1", timeNow.Add(longTTL), timeNow)
	s4Expired := generateAPISession(t, "user1", timeNow.Add(negativeTTL), timeNow)
	s5Expired := generateAPISession(t, "user1", timeNow.Add(negativeTTL), timeNow)

	// check create
	p := newInmemoryDB(t, &s1, &s2, &s3, &s4Expired, &s5Expired)

	s5Updated := APISession{
		SessionID: s5Expired.SessionID,
		ExpiresAt: timeNow.Add(longTTL),
	}

	// check get all(unexpired)
	gotAll1, err := p.GetAll(ctx)

	require.NoError(t, err)
	require.NotEmpty(t, gotAll1)

	assert.Equal(t, s1, gotAll1[0])

	assert.ElementsMatch(t, []APISession{s1, s2, s3}, gotAll1)

	// check expired are in DB
	_, gotExpiredS4, err := p.Get(ctx, s4Expired.SessionID)
	require.NoError(t, err)
	assert.EqualValues(t, s4Expired, gotExpiredS4)
	_, gotExpiredS5, err := p.Get(ctx, s5Expired.SessionID)
	require.NoError(t, err)
	assert.EqualValues(t, s5Expired, gotExpiredS5)

	// check updated
	_, err = p.Save(ctx, s5Updated)
	require.NoError(t, err)

	// check get all(unexpired)
	gotAll2, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []APISession{s1, s2, s3, s5Updated}, gotAll2)

	// check expired is in DB
	_, gotExpired, err := p.Get(ctx, s4Expired.SessionID)
	require.NoError(t, err)
	assert.EqualValues(t, s4Expired, gotExpired)

	// check updated
	_, gotUpdated, err := p.Get(ctx, s5Expired.SessionID)
	require.NoError(t, err)

	assert.EqualValues(t, s5Updated, gotUpdated)

	// check delete by token
	err = p.Delete(ctx, s2.SessionID)
	require.NoError(t, err)
	gotAll3, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []APISession{s1, s3, s5Updated}, gotAll3)

	// make sure expired is in DB
	found, gotExpired2, err := p.Get(ctx, s4Expired.SessionID)
	require.NoError(t, err)
	require.True(t, found)

	assert.EqualValues(t, s4Expired, gotExpired2)

	// check delete all expired
	err = p.DeleteExpired(ctx)
	require.NoError(t, err)
	gotAll4, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []APISession{s1, s3, s5Updated}, gotAll4)

	// make sure expired is not in DB
	found, _, err = p.Get(ctx, s4Expired.SessionID)
	require.NoError(t, err)
	require.False(t, found)
}

func newInmemoryDB(t *testing.T, sessions ...*APISession) *SqliteProvider {
	p, err := NewSqliteProvider(":memory:", DataSourceOptions)
	require.NoError(t, err)

	for _, cur := range sessions {
		sessionID, err := p.Save(context.Background(), *cur)
		require.NoError(t, err)

		// this will patch the supplied sessions to give them their session ids
		cur.SessionID = sessionID
	}

	return p
}

type Token struct {
	Username string `json:"username,omitempty"`
	jwt.StandardClaims
}

func generateAPISession(t *testing.T, username string, expiresAt time.Time, lastAccessAt time.Time) (apiSession APISession) {
	t.Helper()

	apiSession = APISession{
		Username:     username,
		ExpiresAt:    expiresAt,
		LastAccessAt: lastAccessAt,
		UserAgent:    "Chrome",
		IPAddress:    "1.2.3.4",
	}

	return apiSession
}
