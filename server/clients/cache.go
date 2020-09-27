package clients

import "sync"

// ClientCache is a thread-safe in-memory cache.
type ClientCache struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

// NewClientCache returns a thread-safe cache with ID as a key populated with given clients.
func NewClientCache(initClients []*Client) *ClientCache {
	m := make(map[string]*Client, len(initClients))
	for _, v := range initClients {
		m[v.ID] = v
	}
	return &ClientCache{clients: m}
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
