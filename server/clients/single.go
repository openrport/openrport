package clients

import (
	"fmt"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type SingleClient struct {
	log         *chshare.Logger
	credentials string
}

func NewSingleClient(log *chshare.Logger, credentials string) *SingleClient {
	return &SingleClient{
		log:         log,
		credentials: credentials,
	}
}

// GetAll returns a list with a single client credentials.
func (c *SingleClient) GetAll() ([]*Client, error) {
	c.log.Infof("Parsing single client auth credentials...")

	client := &Client{}
	client.ID, client.Password = chshare.ParseAuth(c.credentials)

	if client.ID == "" || client.Password == "" {
		return nil, fmt.Errorf("invalid client auth credentials, expected '<client-id>:<password>', got %q", c.credentials)
	}

	return []*Client{client}, nil
}
