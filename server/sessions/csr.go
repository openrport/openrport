package sessions

import (
	"sync"
	"time"
)

type ClientSessionRepository struct {
	sessions        map[string]*ClientSession
	mu              sync.RWMutex
	KeepLostClients *time.Duration
}

// NewSessionRepository returns a new thread-safe in-memory cache to store Client Sessions populated with given sessions if any.
// keepLostClients is a duration to keep disconnected clients. If a client session was disconnected longer than a given
// duration it will be treated as obsolete.
func NewSessionRepository(initSessions []*ClientSession, keepLostClients *time.Duration) *ClientSessionRepository {
	sessions := make(map[string]*ClientSession)
	for i := range initSessions {
		sessions[initSessions[i].ID] = initSessions[i]
	}
	return &ClientSessionRepository{
		sessions:        sessions,
		KeepLostClients: keepLostClients,
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
		if session.Obsolete(s.KeepLostClients) {
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

// GetActiveByID returns non-obsolete active or disconnected client session by a given id.
func (s *ClientSessionRepository) GetByID(id string) (*ClientSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session := s.sessions[id]
	if session != nil && session.Obsolete(s.KeepLostClients) {
		return nil, nil
	}
	return session, nil
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

// TODO(m-terel): make it consistent with others whether to return an error. In general it's just a cache, so should not return an err.
func (s *ClientSessionRepository) GetAllByClientID(clientID string) []*ClientSession {
	all, _ := s.GetAll()
	var res []*ClientSession
	for _, v := range all {
		if v.ClientID == clientID {
			res = append(res, v)
		}
	}
	return res
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
		if !session.Obsolete(s.KeepLostClients) {
			result = append(result, session)
		}
	}
	return result, nil
}
