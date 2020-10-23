package clients

type Provider interface {
	// Get returns client from provider or nil
	Get(id string) (*Client, error)
	// GetAll returns all clients from provider
	GetAll() ([]*Client, error)
	// Add returns true if the client was added and false if it already exists
	Add(client *Client) (bool, error)
	// Delete returns client by id
	Delete(id string) error
	// IsWriteable returns true if provider is writeable
	IsWriteable() bool
}

// mockProvider is non thread safe in memory provider for use in tests
type mockProvider struct {
	clients map[string]*Client
}

func NewMockProvider(clients []*Client) Provider {
	p := &mockProvider{
		clients: make(map[string]*Client),
	}
	for _, c := range clients {
		p.clients[c.ID] = c
	}
	return p
}

func (p *mockProvider) GetAll() ([]*Client, error) {
	result := make([]*Client, 0, len(p.clients))
	for _, c := range p.clients {
		result = append(result, c)
	}
	return result, nil
}

func (p *mockProvider) Get(id string) (*Client, error) {
	return p.clients[id], nil
}

func (p *mockProvider) Add(client *Client) (bool, error) {
	if _, ok := p.clients[client.ID]; ok {
		return false, nil
	}
	p.clients[client.ID] = client
	return true, nil
}

func (p *mockProvider) Delete(id string) error {
	delete(p.clients, id)
	return nil
}

func (p *mockProvider) IsWriteable() bool {
	return true
}
