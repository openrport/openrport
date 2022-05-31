package clientsauth

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/cloudradar-monitoring/rport/share/enums"
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

func (c *SingleProvider) GetFiltered(filter *query.ListOptions) ([]*ClientAuth, int, error) {
	var ca = []*ClientAuth{c.client}
	if len(filter.Filters) > 0 {
		re := regexp.MustCompile("^" + strings.Replace(filter.Filters[0].Values[0], "*", ".*?", -1) + "$")
		if re.MatchString(ca[0].ID) {
			return ca, 1, nil
		}
		return []*ClientAuth{}, 0, nil
	}
	iOffset, _ := strconv.Atoi(filter.Pagination.Offset)
	if iOffset > 0 {
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
