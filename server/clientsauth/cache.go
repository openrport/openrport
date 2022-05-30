package clientsauth

import (
	"sync"

	"github.com/cloudradar-monitoring/rport/share/enums"
)

// CachedProvider is a thread-safe in-memory cache around the provider.
type CachedProvider struct {
	provider Provider
	clients  map[string]*ClientAuth
	mu       sync.RWMutex
}

var _ Provider = &CachedProvider{}

// NewCachedProvider returns a thread-safe cache around the provider.
func NewCachedProvider(provider Provider) (*CachedProvider, error) {
	clients, err := provider.GetAll()
	if err != nil {
		return nil, err
	}
	m := make(map[string]*ClientAuth, len(clients))
	for _, v := range clients {
		m[v.ID] = v
	}
	return &CachedProvider{
		clients:  m,
		provider: provider,
	}, nil
}

func (c *CachedProvider) Get(id string) (*ClientAuth, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.clients[id], nil
}

func (c *CachedProvider) GetAll() ([]*ClientAuth, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]*ClientAuth, 0, len(c.clients))
	for _, v := range c.clients {
		res = append(res, v)
	}
	return res, nil
}

// Add returns true if a new client auth by a given id was added successfully.
// Returns false if it already contains a client auth with such id.
func (c *CachedProvider) Add(client *ClientAuth) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.clients[client.ID] != nil {
		return false, nil
	}
	_, err := c.provider.Add(client)
	if err != nil {
		return false, err
	}
	c.clients[client.ID] = client
	return true, nil
}

func (c *CachedProvider) Delete(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.provider.Delete(id)
	if err != nil {
		return err
	}
	delete(c.clients, id)
	return nil
}

func (c *CachedProvider) IsWriteable() bool {
	return c.provider.IsWriteable()
}

func (c *CachedProvider) Source() enums.ProviderSource {
	return c.provider.Source()
}
