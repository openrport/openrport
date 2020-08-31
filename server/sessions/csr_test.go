package sessions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSRWithExpiration(t *testing.T) {
	now = nowMockF

	exp := 2 * time.Hour
	repo := NewSessionRepository([]*ClientSession{s1, s2}, &exp)

	assert := assert.New(t)
	assert.NoError(repo.Save(s3))
	assert.NoError(repo.Save(s4))

	gotCount, err := repo.Count()
	assert.NoError(err)
	assert.Equal(3, gotCount)

	gotSessions, err := repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*ClientSession{s1, s2, s3}, gotSessions)

	// active
	gotSession, err := repo.GetActiveByID(s1.ID)
	assert.NoError(err)
	assert.Equal(s1, gotSession)

	// disconnected
	gotSession, err = repo.GetActiveByID(s2.ID)
	assert.NoError(err)
	assert.Nil(gotSession)

	deleted, err := repo.DeleteObsolete()
	assert.NoError(err)
	require.Len(t, deleted, 1)
	assert.Equal(s4, deleted[0])
	gotSessions, err = repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*ClientSession{s1, s2, s3}, gotSessions)

	assert.NoError(repo.Delete(s3))
	gotSessions, err = repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*ClientSession{s1, s2}, gotSessions)
}

func TestCSRWithNoExpiration(t *testing.T) {
	now = nowMockF

	repo := NewSessionRepository([]*ClientSession{s1, s2, s3}, nil)
	s4Active := *s4
	s4Active.Disconnected = nil

	assert := assert.New(t)
	assert.NoError(repo.Save(&s4Active))

	gotCount, err := repo.Count()
	assert.NoError(err)
	assert.Equal(4, gotCount)

	gotSessions, err := repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*ClientSession{s1, s2, s3, &s4Active}, gotSessions)

	// active
	gotSession, err := repo.GetActiveByID(s1.ID)
	assert.NoError(err)
	assert.Equal(s1, gotSession)

	// disconnected
	gotSession, err = repo.GetActiveByID(s2.ID)
	assert.NoError(err)
	assert.Nil(gotSession)

	deleted, err := repo.DeleteObsolete()
	assert.NoError(err)
	assert.Len(deleted, 0)

	assert.NoError(repo.Delete(&s4Active))
	gotSessions, err = repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*ClientSession{s1, s2, s3}, gotSessions)
}
