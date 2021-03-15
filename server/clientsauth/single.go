package clientsauth

import (
	"errors"
)

type SingleProvider struct {
	client *ClientAuth
}

var _ Provider = &SingleProvider{}

func NewSingleProvider(id, password string) *SingleProvider {
	return &SingleProvider{
		client: &ClientAuth{
			ID:       id,
			Password: password,
		},
	}
}

// GetAll returns a list with a single client auth credentials.
func (c *SingleProvider) GetAll() ([]*ClientAuth, error) {
	return []*ClientAuth{c.client}, nil
}

func (c *SingleProvider) Get(id string) (*ClientAuth, error) {
	if c.client.ID == id {
		return c.client, nil
	}
	return nil, nil
}

func (c *SingleProvider) Add(*ClientAuth) (bool, error) {
	return false, errors.New("not implemented")
}

func (c *SingleProvider) Delete(string) error {
	return errors.New("not implemented")
}

func (c *SingleProvider) IsWriteable() bool {
	return false
}

func (c *SingleProvider) Source() ProviderSource {
	return ProviderSourceStatic
}
