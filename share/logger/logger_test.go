package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	logfile := t.TempDir() + "/test.log"
	l, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0444)
	assert.NoError(t, err, "error creating log file")
	defer l.Close()
	logger := NewLogger("test", LogOutput{File: l}, LogLevelDebug)
	logger.Debugf("Debug %s", "Debug")
	logger.Infof("Info %s", "Info")
	logger.Errorf("Error %s", "Error")
	log, err := os.ReadFile(logfile)
	assert.NoError(t, err, "error reading log file")
	assert.Contains(t, string(log), "test: Debug Debug")
	assert.Contains(t, string(log), "test: Info Info")
	assert.Contains(t, string(log), "test: Error Error")
}
