package clientsauth

import (
	"errors"

	"github.com/cloudradar-monitoring/rport/share/enums"
	"github.com/cloudradar-monitoring/rport/share/query"
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

func (c *SingleProvider) GetFiltered(filter *query.ListOptions) ([]*ClientAuth, int, error) {
	var ca = []*ClientAuth{c.client}
	if len(filter.Filters) > 0 {
		match, err := query.MatchesFilters(ca[0], filter.Filters)
		if err != nil {
			return nil, 0, err
		}
		if match {
			return ca, 1, nil
		}
		return []*ClientAuth{}, 0, nil
	}
	start, _ := filter.Pagination.GetStartEnd(1)
	if start > 0 {
		return []*ClientAuth{}, 1, nil
	}
	return ca, 1, nil
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

func (c *SingleProvider) Source() enums.ProviderSource {
	return enums.ProviderSourceStatic
}
