package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SetupTestAPISessionCache(t *testing.T) (ctx context.Context, cache *Cache) {
	t.Helper()

	ctx = context.Background()

	defaultExpiration := time.Duration(100)
	cleanupInterval := time.Duration(200)

	storage, err := NewSqliteProvider(":memory:", DataSourceOptions)
	require.NoError(t, err)

	cache, err = NewCache(ctx, defaultExpiration, cleanupInterval, storage, nil)
	require.NoError(t, err)

	return ctx, cache
}

func TestShouldAddSessionsToCacheAndStorage(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)

	sid, err := c.Save(ctx, s1)
	require.NoError(t, err)
	s1.SessionID = sid

	sid, err = c.Save(ctx, s2)
	require.NoError(t, err)
	s2.SessionID = sid

	// first check the cache
	_, cachedS1, err := c.Get(ctx, s1.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s1, cachedS1)

	_, cachedS2, err := c.Get(ctx, s2.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s2, cachedS2)

	// then check storage
	_, storedS1, err := c.storage.Get(ctx, s1.SessionID)
	require.NoError(t, err)

	// stored version of s1 will have the session id assigned
	s1.SessionID = storedS1.SessionID
	assert.Equal(t, s1, storedS1)

	_, storedS2, err := c.storage.Get(ctx, s2.SessionID)
	require.NoError(t, err)

	// stored version of s2 will have the session id assigned
	s2.SessionID = storedS2.SessionID
	assert.Equal(t, s2, storedS2)
}

func TestShouldGetSortedSessionsForUser(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow.Add(1*time.Minute))
	s3 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)

	sid, err := c.Save(ctx, s1)
	require.NoError(t, err)
	s1.SessionID = sid

	sid, err = c.Save(ctx, s2)
	require.NoError(t, err)
	s2.SessionID = sid

	sid, err = c.Save(ctx, s3)
	require.NoError(t, err)
	s3.SessionID = sid

	sessions, err := c.GetAllByUser(ctx, s2.Username)
	require.NoError(t, err)

	assert.Equal(t, 2, len(sessions))

	// s3 was the oldest, so should be first
	assert.Equal(t, s3, sessions[0])
	assert.Equal(t, s2, sessions[1])
}

func TestShouldUpdateSession(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)

	sid, err := c.Save(ctx, s1)
	require.NoError(t, err)
	s1.SessionID = sid

	sid, err = c.Save(ctx, s2)
	require.NoError(t, err)
	s2.SessionID = sid

	s2.IPAddress = "4.3.2.1"

	_, err = c.Save(ctx, s2)
	require.NoError(t, err)

	// first check the cache
	_, cachedS2, err := c.Get(ctx, s2.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s2.IPAddress, cachedS2.IPAddress)

	// then check storage
	_, storedS2, err := c.storage.Get(ctx, s2.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s2.IPAddress, storedS2.IPAddress)
}

func TestShouldDeleteSessionFromCacheAndStorageByToken(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)

	sid, err := c.Save(ctx, s1)
	require.NoError(t, err)
	s1.SessionID = sid

	sid, err = c.Save(ctx, s2)
	require.NoError(t, err)
	s2.SessionID = sid

	// delete the s2 session
	err = c.Delete(ctx, s2.SessionID)
	require.NoError(t, err)

	// check cached S2 no longer returned
	found, _, err := c.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.False(t, found)

	// check stored S2 no longer returned
	found, _, err = c.storage.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.False(t, found)

	// just as an extra check, check that storedS1 is still there
	found, storedS1, err := c.storage.Get(ctx, s1.SessionID)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, s1, storedS1)
}

func TestShouldDeleteSessionForUser(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)

	sid, err := c.Save(ctx, s1)
	require.NoError(t, err)
	s1.SessionID = sid

	sid, err = c.Save(ctx, s2)
	require.NoError(t, err)
	s2.SessionID = sid

	// delete the s2 session
	err = c.DeleteByID(ctx, s2.Username, s2.SessionID)
	require.NoError(t, err)

	// check cached S2 no longer returned
	found, _, err := c.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.False(t, found)

	// check stored S2 no longer returned
	found, _, err = c.storage.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.False(t, found)

	// just as an extra check, check that storedS1 is still there
	found, storedS1, err := c.storage.Get(ctx, s1.SessionID)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, s1, storedS1)
}

func TestShouldDeleteAllSessionsForUserFromCache(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)
	s3 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)

	sid, err := c.Save(ctx, s1)
	require.NoError(t, err)
	s1.SessionID = sid

	sid, err = c.Save(ctx, s2)
	require.NoError(t, err)
	s2.SessionID = sid

	sid, err = c.Save(ctx, s3)
	require.NoError(t, err)
	s3.SessionID = sid

	// delete sessions for user2
	err = c.DeleteAllByUser(ctx, s2.Username)
	require.NoError(t, err)

	// check cached S2 no longer returned
	found, _, err := c.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.False(t, found)

	// check stored S2 no longer returned
	found, _, err = c.storage.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.False(t, found)

	// check cached S3 no longer returned
	found, _, err = c.Get(ctx, s3.SessionID)
	assert.NoError(t, err)
	assert.False(t, found)

	// check stored S3 no longer returned
	found, _, err = c.storage.Get(ctx, s3.SessionID)
	assert.NoError(t, err)
	assert.False(t, found)

	// just as an extra check, check that storedS1 is still there
	found, storedS1, err := c.storage.Get(ctx, s1.SessionID)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, s1, storedS1)
}

func TestShouldNotChangeExistingSessionsWhenDeleteAllForUnknownUser(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)
	s3 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)

	sid, err := c.Save(ctx, s1)
	require.NoError(t, err)
	s1.SessionID = sid

	sid, err = c.Save(ctx, s2)
	require.NoError(t, err)
	s2.SessionID = sid

	sid, err = c.Save(ctx, s3)
	require.NoError(t, err)
	s3.SessionID = sid

	// delete sessions for unknown user
	err = c.DeleteAllByUser(ctx, "user99")
	require.NoError(t, err)

	// first check the cache
	found, cachedS1, err := c.Get(ctx, s1.SessionID)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, s1, cachedS1)

	found, cachedS2, err := c.Get(ctx, s2.SessionID)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, s2, cachedS2)

	// then check storage
	found, storedS1, err := c.storage.Get(ctx, s1.SessionID)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, s1, storedS1)

	found, storedS2, err := c.storage.Get(ctx, s2.SessionID)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, s2, storedS2)
}
