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

	_, err := c.Save(ctx, s1)
	require.NoError(t, err)

	_, err = c.Save(ctx, s2)
	require.NoError(t, err)

	// first check the cache
	cachedS1, err := c.Get(ctx, s1.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s1, cachedS1)

	cachedS2, err := c.Get(ctx, s2.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s2, cachedS2)

	// then check storage
	storedS1, err := c.storage.Get(ctx, s1.SessionID)
	require.NoError(t, err)

	// stored version of s1 will have the session id assigned
	s1.SessionID = storedS1.SessionID
	assert.Equal(t, s1, storedS1)

	storedS2, err := c.storage.Get(ctx, s2.SessionID)
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

	_, err := c.Save(ctx, s1)
	require.NoError(t, err)

	_, err = c.Save(ctx, s2)
	require.NoError(t, err)

	_, err = c.Save(ctx, s3)
	require.NoError(t, err)

	sessions, err := c.GetAllByUser(ctx, s2.Username)
	assert.NoError(t, err)

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

	_, err := c.Save(ctx, s1)
	require.NoError(t, err)

	_, err = c.Save(ctx, s2)
	require.NoError(t, err)

	s2.IPAddress = "4.3.2.1"

	_, err = c.Save(ctx, s2)
	require.NoError(t, err)

	// first check the cache
	cachedS2, err := c.Get(ctx, s2.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s2.IPAddress, cachedS2.IPAddress)

	// then check storage
	storedS2, err := c.storage.Get(ctx, s2.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s2.IPAddress, storedS2.IPAddress)
}

func TestShouldDeleteSessionFromCacheAndStorageByToken(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)

	_, err := c.Save(ctx, s1)
	require.NoError(t, err)

	_, err = c.Save(ctx, s2)
	require.NoError(t, err)

	// delete the s2 session
	err = c.Delete(ctx, s2.SessionID)
	require.NoError(t, err)

	// check cached S2 no longer returned
	cachedS2, err := c.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.Nil(t, cachedS2)

	// check stored S2 no longer returned
	storedS2, err := c.storage.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.Nil(t, storedS2)

	// just as an extra check, check that storedS1 is still there
	storedS1, err := c.storage.Get(ctx, s1.SessionID)
	assert.NoError(t, err)
	assert.NotNil(t, storedS1)
}

func TestShouldDeleteSessionForUser(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)

	_, err := c.Save(ctx, s1)
	require.NoError(t, err)

	_, err = c.Save(ctx, s2)
	require.NoError(t, err)

	// delete the s2 session
	err = c.DeleteByID(ctx, s2.Username, s2.SessionID)
	require.NoError(t, err)

	// check cached S2 no longer returned
	cachedS2, err := c.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.Nil(t, cachedS2)

	// check stored S2 no longer returned
	storedS2, err := c.storage.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.Nil(t, storedS2)

	// just as an extra check, check that storedS1 is still there
	storedS1, err := c.storage.Get(ctx, s1.SessionID)
	assert.NoError(t, err)
	assert.NotNil(t, storedS1)
}

func TestShouldDeleteAllSessionsForUserFromCache(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)
	s3 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)

	_, err := c.Save(ctx, s1)
	require.NoError(t, err)

	_, err = c.Save(ctx, s2)
	require.NoError(t, err)

	_, err = c.Save(ctx, s3)
	require.NoError(t, err)

	// delete sessions for user2
	err = c.DeleteAllByUser(ctx, s2.Username)
	require.NoError(t, err)

	// check cached S2 no longer returned
	cachedS2, err := c.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.Nil(t, cachedS2)

	// check stored S2 no longer returned
	storedS2, err := c.storage.Get(ctx, s2.SessionID)
	assert.NoError(t, err)
	assert.Nil(t, storedS2)

	// check cached S3 no longer returned
	cachedS3, err := c.Get(ctx, s3.SessionID)
	assert.NoError(t, err)
	assert.Nil(t, cachedS3)

	// check stored S3 no longer returned
	storedS3, err := c.storage.Get(ctx, s3.SessionID)
	assert.NoError(t, err)
	assert.Nil(t, storedS3)

	// just as an extra check, check that storedS1 is still there
	storedS1, err := c.storage.Get(ctx, s1.SessionID)
	assert.NoError(t, err)
	assert.NotNil(t, storedS1)
}

func TestShouldNotChangeExistingSessionsWhenDeleteAllForUnknownUser(t *testing.T) {
	ctx, c := SetupTestAPISessionCache(t)

	longTTL := time.Hour
	timeNow := time.Now().UTC()

	s1 := generateAPISession(t, "user1", timeNow.UTC().Add(longTTL), timeNow)
	s2 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)
	s3 := generateAPISession(t, "user2", timeNow.UTC().Add(longTTL), timeNow)

	_, err := c.Save(ctx, s1)
	require.NoError(t, err)

	_, err = c.Save(ctx, s2)
	require.NoError(t, err)

	_, err = c.Save(ctx, s3)
	require.NoError(t, err)

	// delete sessions for unknown user
	err = c.DeleteAllByUser(ctx, "user99")
	require.NoError(t, err)

	// first check the cache
	cachedS1, err := c.Get(ctx, s1.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s1, cachedS1)

	cachedS2, err := c.Get(ctx, s2.SessionID)
	require.NoError(t, err)
	assert.Equal(t, s2, cachedS2)

	// then check storage
	storedS1, err := c.storage.Get(ctx, s1.SessionID)
	require.NoError(t, err)

	// stored version of s1 will have the session id assigned
	s1.SessionID = storedS1.SessionID
	assert.Equal(t, s1, storedS1)

	storedS2, err := c.storage.Get(ctx, s2.SessionID)
	require.NoError(t, err)

	// stored version of s2 will have the session id assigned
	s2.SessionID = storedS2.SessionID
	assert.Equal(t, s2, storedS2)
}
