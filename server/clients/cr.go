package clients

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ClientRepository struct {
	// in-memory cache
	clients         map[string]*Client
	mu              sync.RWMutex
	KeepLostClients *time.Duration
	// storage
	provider ClientProvider
}

// NewClientRepository returns a new thread-safe in-memory cache to store client connections populated with given clients if any.
// keepLostClients is a duration to keep disconnected clients. If a client was disconnected longer than a given
// duration it will be treated as obsolete.
func NewClientRepository(initClients []*Client, keepLostClients *time.Duration) *ClientRepository {
	return newClientRepositoryWithDB(initClients, keepLostClients, nil)
}

func newClientRepositoryWithDB(initClients []*Client, keepLostClients *time.Duration, provider ClientProvider) *ClientRepository {
	clients := make(map[string]*Client)
	for i := range initClients {
		clients[initClients[i].ID] = initClients[i]
	}
	return &ClientRepository{
		clients:         clients,
		KeepLostClients: keepLostClients,
		provider:        provider,
	}
}

func InitClientRepository(ctx context.Context, provider ClientProvider, keepLostClients *time.Duration) (*ClientRepository, error) {
	initClients, err := GetInitState(ctx, provider)
	if err != nil {
		return nil, err
	}

	return newClientRepositoryWithDB(initClients, keepLostClients, provider), nil
}

func (s *ClientRepository) Save(client *Client) error {
	if s.provider != nil {
		err := s.provider.Save(context.Background(), client)
		if err != nil {
			return fmt.Errorf("failed to save a client: %w", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client.ID] = client
	return nil
}

func (s *ClientRepository) Delete(client *Client) error {
	if s.provider != nil {
		err := s.provider.Delete(context.Background(), client.ID)
		if err != nil {
			return fmt.Errorf("failed to delete a client: %w", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, client.ID)
	return nil
}

// DeleteObsolete deletes obsolete disconnected clients and returns them.
func (s *ClientRepository) DeleteObsolete() ([]*Client, error) {
	if s.provider != nil {
		err := s.provider.DeleteObsolete(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to delete obsolete clients: %w", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	var deleted []*Client
	for _, client := range s.clients {
		if client.Obsolete(s.KeepLostClients) {
			delete(s.clients, client.ID)
			deleted = append(deleted, client)
		}
	}
	return deleted, nil
}

// Count returns a number of non-obsolete active and disconnected clients.
func (s *ClientRepository) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clients, err := s.getNonObsolete()
	return len(clients), err
}

// CountActive returns a number of active clients.
func (s *ClientRepository) CountActive() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.GetAllActive()), nil
}

// CountDisconnected returns a number of disconnected clients.
func (s *ClientRepository) CountDisconnected() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all, err := s.getNonObsolete()
	if err != nil {
		return 0, err
	}

	var n int
	for _, cur := range all {
		if cur.DisconnectedAt != nil {
			n++
		}
	}
	return n, nil
}

// GetActiveByID returns non-obsolete active or disconnected client by a given id.
func (s *ClientRepository) GetByID(id string) (*Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client := s.clients[id]
	if client != nil && client.Obsolete(s.KeepLostClients) {
		return nil, nil
	}
	return client, nil
}

// GetActiveByID returns an active client by a given id.
func (s *ClientRepository) GetActiveByID(id string) (*Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client := s.clients[id]
	if client != nil && client.DisconnectedAt != nil {
		return nil, nil
	}
	return client, nil
}

// TODO(m-terel): make it consistent with others whether to return an error. In general it's just a cache, so should not return an err.
func (s *ClientRepository) GetAllByClientAuthID(clientAuthID string) []*Client {
	all, _ := s.GetAll()
	var res []*Client
	for _, v := range all {
		if v.ClientAuthID == clientAuthID {
			res = append(res, v)
		}
	}
	return res
}

// GetAll returns all non-obsolete active and disconnected client clients.
func (s *ClientRepository) GetAll() ([]*Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getNonObsolete()
}

func (s *ClientRepository) GetAllActive() []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Client
	for _, client := range s.clients {
		if client.DisconnectedAt == nil {
			result = append(result, client)
		}
	}
	return result
}

func (s *ClientRepository) getNonObsolete() ([]*Client, error) {
	result := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		if !client.Obsolete(s.KeepLostClients) {
			result = append(result, client)
		}
	}
	return result, nil
}
