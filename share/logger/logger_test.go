package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	logfile := t.TempDir() + "/test.log"
	l, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0444)
	require.NoError(t, err, "error creating log file")
	defer l.Close()

	logger := NewLogger("test", LogOutput{File: l}, LogLevelDebug)
	logger.Debugf("Debug %s", "Debug")
	logger.Infof("Info %s", "Info")
	logger.Errorf("Error %s", "Error")

	log, err := os.ReadFile(logfile)

	require.NoError(t, err, "error reading log file")
	assert.Contains(t, string(log), "debug: test: Debug Debug")
	assert.Contains(t, string(log), "info: test: Info Info")
	assert.Contains(t, string(log), "error: test: Error Error")
}
