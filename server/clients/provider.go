package clients

type Provider interface {
	GetAll() ([]*Client, error)
}

type mockProvider struct {
	clients []*Client
}

func NewMockProvider(clients []*Client) Provider {
	return &mockProvider{
		clients: clients,
	}
}

func (p *mockProvider) GetAll() ([]*Client, error) {
	return p.clients, nil
}
