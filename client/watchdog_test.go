package chclient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openrport/openrport/share/logger"
)

func TestWatchdog(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	testLog := logger.NewLogger("client", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	_, err := NewWatchdog(false, dir, testLog)
	require.NoError(t, err)
	w, err := NewWatchdog(true, dir, testLog)
	require.NoError(t, err)
	testCases := []struct {
		state    string
		msg      string
		expected string
	}{
		{
			state:    WatchdogStateInit,
			expected: "initialized",
		},
		{
			state:    WatchdogStateReconnecting,
			expected: "reconnecting",
			msg:      "some error",
		},
		{
			state:    WatchdogStateConnected,
			expected: "connected",
			msg:      "connected to 127.0.0.1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.state, func(t *testing.T) {
			w.Ping(tc.state, tc.msg)
			s, err := os.ReadFile(stateFile)
			require.NoError(t, err)
			var ws watchdogState
			err = json.Unmarshal(s, &ws)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, ws.LastState)
			assert.Equal(t, tc.msg, ws.LastMessage)
			assert.WithinDurationf(t, ws.LastUpdate, time.Now(), 10*time.Millisecond, "")
		})
	}
}
