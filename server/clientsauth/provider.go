package clientsauth

import (
	"github.com/realvnc-labs/rport/share/enums"
	"github.com/realvnc-labs/rport/share/query"
)

type Provider interface {
	// Get returns client authentication credentials from provider or nil
	Get(id string) (*ClientAuth, error)
	// GetFiltered returns authentication credentials and total count filtered
	GetFiltered(filter *query.ListOptions) ([]*ClientAuth, int, error)
	// Add returns true if the client auth was added and false if it already exists
	Add(client *ClientAuth) (bool, error)
	// Delete returns client auth by id
	Delete(id string) error
	// IsWriteable returns true if provider is writeable
	IsWriteable() bool
	// Source returns a provider source
	Source() enums.ProviderSource
}
