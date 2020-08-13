package chserver

type ClientSessionRepository struct {
	sessions map[string]*ClientSession
}

func NewSessionRepository() *ClientSessionRepository {
	return &ClientSessionRepository{
		sessions: make(map[string]*ClientSession),
	}
}

func (s *ClientSessionRepository) Save(session *ClientSession) error {
	s.sessions[session.ID] = session
	return nil
}

func (s *ClientSessionRepository) Delete(session *ClientSession) error {
	delete(s.sessions, session.ID)
	return nil
}

func (s *ClientSessionRepository) Count() (int, error) {
	return len(s.sessions), nil
}

func (s *ClientSessionRepository) FindOne(id string) (*ClientSession, error) {
	c, exists := s.sessions[id]
	if !exists {
		return nil, nil
	}
	return c, nil
}

func (s *ClientSessionRepository) GetAll() ([]*ClientSession, error) {
	var result = make([]*ClientSession, 0)
	for _, c := range s.sessions {
		result = append(result, c)
	}
	return result, nil
}
