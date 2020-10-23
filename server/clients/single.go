package clients

import (
	"errors"
)

type SingleProvider struct {
	client *Client
}

func NewSingleProvider(id, password string) *SingleProvider {
	return &SingleProvider{
		client: &Client{
			ID:       id,
			Password: password,
		},
	}
}

// GetAll returns a list with a single client credentials.
func (c *SingleProvider) GetAll() ([]*Client, error) {
	return []*Client{c.client}, nil
}

func (c *SingleProvider) Get(id string) (*Client, error) {
	if c.client.ID == id {
		return c.client, nil
	}
	return nil, nil
}

func (c *SingleProvider) Add(*Client) (bool, error) {
	return false, errors.New("not implemented")
}

func (c *SingleProvider) Delete(string) error {
	return errors.New("not implemented")
}

func (c *SingleProvider) IsWriteable() bool {
	return false
}
