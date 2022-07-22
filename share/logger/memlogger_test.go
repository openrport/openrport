package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestMemLogger(t *testing.T) {
	mLog := NewMemLogger()
	mLog.Debugf("Debug %s", "Debug")
	mLog.Infof("Info %s", "Info")
	mLog.Errorf("Error %s", "Error")
	logfile := t.TempDir() + "/test.log"
	l, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0444)
	require.NoError(t, err, "error creating log file")
	defer l.Close()
	mLog.Flush(NewLogger("test", LogOutput{File: l}, LogLevelDebug))
	log, err := os.ReadFile(logfile)
	assert.NoError(t, err, "error reading log file")
	assert.Contains(t, string(log), "test: Debug Debug")
	assert.Contains(t, string(log), "test: Info Info")
	assert.Contains(t, string(log), "test: Error Error")
}
