package users

import (
	"errors"

	"github.com/cloudradar-monitoring/rport/share/enums"
)

type StaticProvider struct {
	*UserCache
}

func NewStaticProvider(users []*User) *StaticProvider {
	return &StaticProvider{
		UserCache: NewUserCache(users),
	}
}

func (p StaticProvider) Type() enums.ProviderSource {
	return enums.ProviderSourceStatic
}

func (p *StaticProvider) GetAllGroups() ([]string, error) {
	return nil, errors.New("not implemented")
}

func (p *StaticProvider) Add(usr *User) error {
	return errors.New("operation not supported for single user authentication")
}
func (p *StaticProvider) Update(usr *User, username string) error {
	return errors.New("operation not supported for single user authentication")
}
func (p *StaticProvider) Delete(username string) error {
	return errors.New("operation not supported for single user authentication")
}
