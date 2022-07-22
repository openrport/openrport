package logger

import (
	"fmt"
	"sync"
)

type MemLogger struct {
	debugMsgs []string
	infoMsgs  []string
	errorMsgs []string
	mu        sync.RWMutex
}

// NewMemLogger creates a logger that stores messages in memory
// used for early logging while the "real" file-based logger is not loaded yet.
func NewMemLogger() MemLogger {
	return MemLogger{}
}
func (ml *MemLogger) Debug(msg string) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.debugMsgs = append(ml.debugMsgs, msg)
}
func (ml *MemLogger) Debugf(msg string, args ...interface{}) {
	ml.Debug(fmt.Sprintf(msg, args...))
}
func (ml *MemLogger) Info(msg string) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.infoMsgs = append(ml.infoMsgs, msg)
}
func (ml *MemLogger) Infof(msg string, args ...interface{}) {
	ml.Info(fmt.Sprintf(msg, args...))
}
func (ml *MemLogger) Error(msg string) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.errorMsgs = append(ml.errorMsgs, msg)
}
func (ml *MemLogger) Errorf(msg string, args ...interface{}) {
	ml.Error(fmt.Sprintf(msg, args...))
}
func (ml *MemLogger) Flush(l *Logger) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	var m string
	for _, m = range ml.debugMsgs {
		l.Debugf(m)
	}
	ml.debugMsgs = nil
	for _, m = range ml.infoMsgs {
		l.Infof(m)
	}
	ml.infoMsgs = nil
	for _, m = range ml.errorMsgs {
		l.Errorf(m)
	}
	ml.errorMsgs = nil
}
