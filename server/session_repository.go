package chserver

type SessionRepository struct {
	sessions map[string]*ClientSession
}

func NewSessionRepository() *SessionRepository {
	return &SessionRepository{
		sessions: make(map[string]*ClientSession),
	}
}

func (s *SessionRepository) Add(session *ClientSession) {
	s.sessions[session.ID] = session
}

func (s *SessionRepository) Delete(session *ClientSession) {
	delete(s.sessions, session.ID)
}

func (s *SessionRepository) Count() int {
	return len(s.sessions)
}

func (s *SessionRepository) FindOne(id string) *ClientSession {
	c, exists := s.sessions[id]
	if !exists {
		return nil
	}
	return c
}

func (s *SessionRepository) GetAll() []*ClientSession {
	var result = make([]*ClientSession, 0)
	for _, c := range s.sessions {
		result = append(result, c)
	}
	return result
}
