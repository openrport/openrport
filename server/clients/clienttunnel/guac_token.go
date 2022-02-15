package clienttunnel

import (
	"sync"
)

type GuacToken struct {
	username     string
	password     string
	security     string
	width        string
	height       string
	serverLayout string
}

type GuacTokenStore struct {
	sync.RWMutex
	GuacTokens map[string]*GuacToken
}

func NewGuacTokenStore() *GuacTokenStore {
	return &GuacTokenStore{
		GuacTokens: map[string]*GuacToken{},
	}
}

func (s *GuacTokenStore) Add(uuid string, token *GuacToken) {
	s.Lock()
	defer s.Unlock()
	s.GuacTokens[uuid] = token
}

func (s *GuacTokenStore) Get(uuid string) *GuacToken {
	s.RLock()
	defer s.RUnlock()
	return s.GuacTokens[uuid]
}

func (s *GuacTokenStore) Delete(uuid string) {
	s.Lock()
	defer s.Unlock()
	delete(s.GuacTokens, uuid)
}
