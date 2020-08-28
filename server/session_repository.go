package chserver

import (
	"sync"
	"time"
)

// TODO(mterel): move to csr package
type ClientSessionRepository struct {
	sessions        map[string]*ClientSession
	mu              sync.RWMutex
	keepLostClients *time.Duration
}

func NewSessionRepository(initSessions []*ClientSession, keepLostClients *time.Duration) *ClientSessionRepository {
	sessions := make(map[string]*ClientSession)
	for i := range initSessions {
		sessions[initSessions[i].ID] = initSessions[i]
	}
	return &ClientSessionRepository{
		sessions:        sessions,
		keepLostClients: keepLostClients,
	}
}

func (s *ClientSessionRepository) Save(session *ClientSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *ClientSessionRepository) Delete(session *ClientSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, session.ID)
	return nil
}

// DeleteObsolete deletes obsolete disconnected client sessions and returns them.
func (s *ClientSessionRepository) DeleteObsolete() ([]*ClientSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var deleted []*ClientSession
	for _, session := range s.sessions {
		if session.Obsolete(s.keepLostClients) {
			delete(s.sessions, session.ID)
			deleted = append(deleted, session)
		}
	}
	return deleted, nil
}

// Count returns a number of non-obsolete active and disconnected client sessions.
func (s *ClientSessionRepository) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessions, err := s.getNonObsolete()
	return len(sessions), err
}

// GetActiveByID returns an active client session by a given id.
func (s *ClientSessionRepository) GetActiveByID(id string) (*ClientSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session := s.sessions[id]
	if session != nil && session.Disconnected != nil {
		return nil, nil
	}
	return session, nil
}

// GetAll returns all non-obsolete active and disconnected client sessions.
func (s *ClientSessionRepository) GetAll() ([]*ClientSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getNonObsolete()
}

func (s *ClientSessionRepository) getNonObsolete() ([]*ClientSession, error) {
	result := make([]*ClientSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		if !session.Obsolete(s.keepLostClients) {
			result = append(result, session)
		}
	}
	return result, nil
}
