package chserver

type SessionRepository struct {
	sessions map[string]*ClientSession
}

func NewSessionRepository() *SessionRepository {
	return &SessionRepository{
		sessions: make(map[string]*ClientSession),
	}
}

func (s *SessionRepository) Save(session *ClientSession) error {
	s.sessions[session.ID] = session
	return nil
}

func (s *SessionRepository) Delete(session *ClientSession) error {
	delete(s.sessions, session.ID)
	return nil
}

func (s *SessionRepository) Count() (int, error) {
	return len(s.sessions), nil
}

func (s *SessionRepository) FindOne(id string) (*ClientSession, error) {
	c, exists := s.sessions[id]
	if !exists {
		return nil, nil
	}
	return c, nil
}

func (s *SessionRepository) GetAll() ([]*ClientSession, error) {
	var result = make([]*ClientSession, 0)
	for _, c := range s.sessions {
		result = append(result, c)
	}
	return result, nil
}
