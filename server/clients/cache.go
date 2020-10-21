package clients

import (
	"errors"
	"sync"
)

// ClientCache is a thread-safe in-memory cache.
type ClientCache struct {
	provider Provider
	clients  map[string]*Client
	mu       sync.RWMutex
}

// NewClientCache returns a thread-safe cache with ID as a key populated with given clients.
func NewClientCache(provider Provider) (*ClientCache, error) {
	clients, err := provider.GetAll()
	if err != nil {
		return nil, err
	}
	m := make(map[string]*Client, len(clients))
	for _, v := range clients {
		m[v.ID] = v
	}
	return &ClientCache{
		clients:  m,
		provider: provider,
	}, nil
}

// NewEmptyClientCache returns a thread-safe empty client cache.
func NewEmptyClientCache() *ClientCache {
	return &ClientCache{clients: map[string]*Client{}}
}

func (c *ClientCache) Get(key string) *Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.clients[key]
}

func (c *ClientCache) GetAll() []*Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]*Client, 0, len(c.clients))
	for _, v := range c.clients {
		res = append(res, v)
	}
	return res
}

func (c *ClientCache) Set(key string, client *Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients[key] = client
}

// Add returns true if a new client by a given key was added successfully.
// Returns false if it already contains a client with such key.
func (c *ClientCache) Add(key string, client *Client) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.clients[key] != nil {
		return false
	}
	c.clients[key] = client
	return true
}

// Lock locks a client with a given client session ID (sid), if it's unlocked or obtained by the same sid and returns true.
// Returns false if it's already locked or if it's failed to do it.
func (c *ClientCache) LockWith(key, sid string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	client := c.clients[key]
	if client == nil {
		return false, errors.New("not found")
	}

	if client.lockedBySID == sid || client.lockedBySID == "" {
		client.lockedBySID = sid
		return true, nil
	}

	return false, nil
}

// Unlock unlocks a client by a given key if it was locked and returns nil.
// Returns an error if it's already unlocked or if it's failed to do it.
func (c *ClientCache) Unlock(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	client := c.clients[key]
	if client == nil {
		return errors.New("not found")
	}

	if client.lockedBySID == "" {
		return errors.New("already unlocked")
	}

	client.lockedBySID = ""
	return nil
}

// Unlock unlocks clients only if client session ids are the same.
func (c *ClientCache) UnlockClientsLockedBySIDs(sidClientPairs map[string]string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var unlockedClients []string
	for sid, clientID := range sidClientPairs {
		client := c.clients[clientID]
		if client == nil {
			// client can be removed, so it's ok to have nil client
			continue
		}

		// unlock only if it was obtained by a current sid
		if client.lockedBySID == sid {
			client.lockedBySID = ""
			unlockedClients = append(unlockedClients, clientID)
		}
	}
	return unlockedClients
}

func (c *ClientCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.clients, key)
}

func (c *ClientCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.clients)
}

func (c *ClientCache) IsSingleClient() bool {
	var i interface{} = c.provider
	switch i.(type) {
	case *SingleClient:
		return true
	}

	return false
}
