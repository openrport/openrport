package session

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPISessionSqlite(t *testing.T) {
	ctx := context.Background()

	longTTL := time.Hour
	negativeTTL := -time.Minute

	s1 := generateAPISession(t, longTTL)
	s2 := generateAPISession(t, longTTL)
	s3 := generateAPISession(t, longTTL)
	s4Expired := generateAPISession(t, negativeTTL)
	s5Expired := generateAPISession(t, negativeTTL)
	s5Updated := &APISession{
		Token:     s5Expired.Token,
		ExpiresAt: time.Now().UTC().Add(longTTL),
	}

	// check create
	p := newInmemoryDB(t, s1, s2, s3, s4Expired, s5Expired)

	// check get all(unexpired)
	gotAll1, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*APISession{s1, s2, s3}, gotAll1)
	// check expired are in DB
	gotExpiredS4, err := p.Get(ctx, s4Expired.Token)
	require.NoError(t, err)
	assert.EqualValues(t, s4Expired, gotExpiredS4)
	gotExpiredS5, err := p.Get(ctx, s5Expired.Token)
	require.NoError(t, err)
	assert.EqualValues(t, s5Expired, gotExpiredS5)

	// check update
	err = p.Save(ctx, s5Updated)
	require.NoError(t, err)

	// check get all(unexpired)
	gotAll2, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*APISession{s1, s2, s3, s5Updated}, gotAll2)
	// check expired is in DB
	gotExpired, err := p.Get(ctx, s4Expired.Token)
	require.NoError(t, err)
	assert.EqualValues(t, s4Expired, gotExpired)
	// check updated
	gotUpdated, err := p.Get(ctx, s5Expired.Token)
	require.NoError(t, err)
	assert.EqualValues(t, s5Updated, gotUpdated)

	// check delete by token
	err = p.Delete(ctx, s2.Token)
	require.NoError(t, err)
	gotAll3, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*APISession{s1, s3, s5Updated}, gotAll3)

	// make sure expired is in DB
	gotExpired2, err := p.Get(ctx, s4Expired.Token)
	require.NoError(t, err)
	require.NotNil(t, gotExpired2)
	// check delete all expired
	err = p.DeleteExpired(ctx)
	require.NoError(t, err)
	gotAll4, err := p.GetAll(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*APISession{s1, s3, s5Updated}, gotAll4)
	// make sure expired is not in DB
	gotExpired3, err := p.Get(ctx, s4Expired.Token)
	require.NoError(t, err)
	require.Nil(t, gotExpired3)
}

func newInmemoryDB(t *testing.T, sessions ...*APISession) *SqliteProvider {
	p, err := NewSqliteProvider(":memory:")
	require.NoError(t, err)

	for _, cur := range sessions {
		require.NoError(t, p.Save(context.Background(), cur))
	}

	return p
}

type Token struct {
	Username string `json:"username,omitempty"`
	jwt.StandardClaims
}

func generateAPISession(t *testing.T, ttl time.Duration) *APISession {
	jwtSecret, err := generateJWTSecret()
	require.NoError(t, err)

	claims := Token{
		Username: "username",
		StandardClaims: jwt.StandardClaims{
			Id: strconv.FormatUint(rand.Uint64(), 10),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	return &APISession{
		Token:     tokenStr,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
}

func generateJWTSecret() (string, error) {
	data := make([]byte, 10)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("can't generate API JWT secret: %s", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
