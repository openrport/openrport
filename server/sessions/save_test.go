package sessions

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSaveToFile(t *testing.T) {
	Now = nowMockF

	// given
	var fileMock bytes.Buffer
	exp := 2 * time.Hour
	// keepLostClients 2 hours: 1 active (s1), 2 disconnected(s2, s3), 1 obsolete(s4)
	sessions := []*ClientSession{s1, s2, s3, s4}
	task := NewSaveToFileTask(testLog, NewSessionRepository(sessions, &exp), "test-file")

	// when
	err := task.getAndSave(&fileMock)

	// then
	assert.NoError(t, err)

	// keepLostClients 1 hour: 1 active (s1), 1 disconnected(s2), 1 obsolete(s3)
	gotSessions, err := getInitState(&fileMock, &hour)
	assert.NoError(t, err)
	wantS1 := shallowCopy(s1)
	wantS1.Disconnected = &nowMock
	assert.ElementsMatch(t, gotSessions, []*ClientSession{wantS1, s2})
}
