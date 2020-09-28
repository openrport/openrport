package sessions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanup(t *testing.T) {
	Now = nowMockF

	// given
	sessions := []*ClientSession{s1, s2, s3}
	repo := NewSessionRepository(sessions, &hour)
	require.Len(t, repo.sessions, 3)
	task := NewCleanupTask(testLog, repo, nil)

	// when
	err := task.Run()

	// then
	assert.NoError(t, err)
	assert.ElementsMatch(t, getValues(repo.sessions), []*ClientSession{s1, s2})
}

func getValues(sessions map[string]*ClientSession) []*ClientSession {
	var r []*ClientSession
	for _, v := range sessions {
		r = append(r, v)
	}
	return r
}
