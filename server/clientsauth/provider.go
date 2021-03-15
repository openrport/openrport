package clientsauth

type ProviderSource string

const (
	ProviderSourceStatic ProviderSource = "Static Credentials"
	ProviderSourceFile   ProviderSource = "File"
	ProviderSourceDB     ProviderSource = "DB"
	ProviderSourceMock   ProviderSource = "Mock"
)

type Provider interface {
	// Get returns client authentication credentials from provider or nil
	Get(id string) (*ClientAuth, error)
	// GetAll returns authentication credentials of all clients from provider
	GetAll() ([]*ClientAuth, error)
	// Add returns true if the client auth was added and false if it already exists
	Add(client *ClientAuth) (bool, error)
	// Delete returns client auth by id
	Delete(id string) error
	// IsWriteable returns true if provider is writeable
	IsWriteable() bool
	// Source returns a provider source
	Source() ProviderSource
}

// mockProvider is non thread safe in memory provider for use in tests
type mockProvider struct {
	clients map[string]*ClientAuth
}

var _ Provider = &mockProvider{}

func NewMockProvider(clients []*ClientAuth) Provider {
	p := &mockProvider{
		clients: make(map[string]*ClientAuth),
	}
	for _, c := range clients {
		p.clients[c.ID] = c
	}
	return p
}

func (p *mockProvider) GetAll() ([]*ClientAuth, error) {
	result := make([]*ClientAuth, 0, len(p.clients))
	for _, c := range p.clients {
		result = append(result, c)
	}
	return result, nil
}

func (p *mockProvider) Get(id string) (*ClientAuth, error) {
	return p.clients[id], nil
}

func (p *mockProvider) Add(client *ClientAuth) (bool, error) {
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

func (p *mockProvider) Source() ProviderSource {
	return ProviderSourceMock
}
