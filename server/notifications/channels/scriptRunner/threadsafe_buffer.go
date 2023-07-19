package scriptRunner

import (
	"bytes"
	"sync"
	"sync/atomic"
)

// Buffer is a goroutine safe bytes.Buffer
type Buffer struct {
	buffer         bytes.Buffer
	mutex          sync.Mutex
	stopCollecting atomic.Bool
}

func (s *Buffer) Write(p []byte) (n int, err error) {
	if s.stopCollecting.Load() {
		return len(p), nil
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Write(p)
}

func (s *Buffer) CollectiongDone() {
	s.stopCollecting.Store(true)
}

func (s *Buffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.String()
}

func (s *Buffer) Len() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Len()
}
