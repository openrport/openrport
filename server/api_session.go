package chserver

import (
	"sync"
	"time"
)

type APISession struct {
	Token     string
	ExpiresAt time.Time
}

type APISessionRepository struct {
	sessions map[string]*APISession
	mu       sync.RWMutex
}

func NewAPISessionRepository() *APISessionRepository {
	return &APISessionRepository{
		sessions: make(map[string]*APISession),
	}
}

func (r *APISessionRepository) Save(session *APISession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.Token] = session
	return nil
}

func (r *APISessionRepository) Delete(session *APISession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, session.Token)
	return nil
}

func (r *APISessionRepository) FindOne(id string) (*APISession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, exists := r.sessions[id]
	if !exists {
		return nil, nil
	}
	return c, nil
}
