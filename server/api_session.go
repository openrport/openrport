package chserver

import "time"

type APISession struct {
	Token     string
	ExpiresAt time.Time
}

type APISessionRepository struct {
	sessions map[string]*APISession
}

func NewAPISessionRepository() *APISessionRepository {
	return &APISessionRepository{
		sessions: make(map[string]*APISession),
	}
}

func (r *APISessionRepository) Save(session *APISession) error {
	r.sessions[session.Token] = session
	return nil
}

func (r *APISessionRepository) Delete(session *APISession) error {
	delete(r.sessions, session.Token)
	return nil
}

func (r *APISessionRepository) FindOne(id string) (*APISession, error) {
	c, exists := r.sessions[id]
	if !exists {
		return nil, nil
	}
	return c, nil
}
