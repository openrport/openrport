package usercli

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/chconfig"
	"github.com/cloudradar-monitoring/rport/share/enums"
)

type UserInput struct {
	Username    string
	Password    string
	Groups      []string
	TwoFASendTo string
}

type UserService interface {
	Create(input UserInput) error
	Change(input UserInput) error
	Delete(username string) error
	ProviderType() enums.ProviderSource
}

func NewUserService(cfg *chconfig.Config) (UserService, error) {
	var authDB *sqlx.DB
	var err error
	if cfg.Database.Driver != "" {
		authDB, err = sqlx.Connect(cfg.Database.Driver, cfg.Database.Dsn)
		if err != nil {
			return nil, fmt.Errorf("Could not connect to user database: %w", err)
		}
	}

	apiUserService, err := users.NewAPIServiceFromConfig(authDB, cfg)
	if err != nil {
		return nil, fmt.Errorf("Could not initialize user service: %w", err)
	}

	return &userServiceImpl{
		apiUserService: apiUserService,
	}, nil
}

type userServiceImpl struct {
	apiUserService *users.APIService
}

func (u *userServiceImpl) Create(input UserInput) error {
	return u.apiUserService.Change(&users.User{
		Username:    input.Username,
		Password:    input.Password,
		Groups:      input.Groups,
		TwoFASendTo: input.TwoFASendTo,
	}, "")
}

func (u *userServiceImpl) Change(input UserInput) error {
	return u.apiUserService.Change(&users.User{
		Password:    input.Password,
		Groups:      input.Groups,
		TwoFASendTo: input.TwoFASendTo,
	}, input.Username)
}

func (u *userServiceImpl) Delete(username string) error {
	return u.apiUserService.Delete(username)
}

func (u *userServiceImpl) ProviderType() enums.ProviderSource {
	return u.apiUserService.GetProviderType()
}
