package users

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/share/collections"
	"github.com/cloudradar-monitoring/rport/share/enums"
)

type Provider interface {
	Type() enums.ProviderSource
	GetAll() ([]*User, error)
	GetAllGroups() ([]string, error)
	GetByUsername(username string) (*User, error)
	Add(usr *User) error
	Update(usr *User, usernameToUpdate string) error
	Delete(usernameToDelete string) error
}

type APIService struct {
	DeliverySrv message.Service
	Provider    Provider
	TwoFAOn     bool
}

func NewAPIService(provider Provider, twoFAOn bool) *APIService {
	return &APIService{
		Provider: provider,
		TwoFAOn:  twoFAOn,
	}
}

func (as APIService) GetProviderType() enums.ProviderSource {
	return as.Provider.Type()
}

func (as *APIService) GetAll() ([]*User, error) {
	return as.Provider.GetAll()
}

func (as *APIService) GetByUsername(username string) (*User, error) {
	return as.Provider.GetByUsername(username)
}

func (as *APIService) GetAllGroups() ([]string, error) {
	return as.Provider.GetAllGroups()
}

func (as *APIService) ExistGroups(groups []string) error {
	existingGroups, err := as.GetAllGroups()
	if err != nil {
		return err
	}

	groupMap := collections.ConvertToStringBoolMap(existingGroups)
	var groupsNotFound []string
	for _, cur := range groups {
		if !groupMap.Has(cur) {
			groupsNotFound = append(groupsNotFound, cur)
		}
	}

	if len(groupsNotFound) > 0 {
		return errors2.APIError{
			Message:    fmt.Sprintf("user groups not found: %v", strings.Join(groupsNotFound, ", ")),
			HTTPStatus: http.StatusNotFound,
		}
	}

	return nil
}

func (as *APIService) Change(usr *User, username string) error {
	if usr.Password != "" {
		passHash, err := bcrypt.GenerateFromPassword([]byte(usr.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		usr.Password = strings.Replace(string(passHash), htpasswdBcryptAltPrefix, htpasswdBcryptPrefix, 1)
	}

	if usr.Token != nil && *usr.Token != "" {
		tokenHash, err := bcrypt.GenerateFromPassword([]byte(*usr.Token), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		tokenHashStr := strings.Replace(string(tokenHash), htpasswdBcryptAltPrefix, htpasswdBcryptPrefix, 1)
		usr.Token = &tokenHashStr
	}

	err := as.validate(usr, username)
	if err != nil {
		return err
	}

	if username != "" {
		return as.updateUser(usr, username)
	}
	return as.addUser(usr)
}

func (as *APIService) validate(dataToChange *User, usernameToFind string) error {
	errs := errors2.APIErrors{}

	if usernameToFind == "" {
		if dataToChange.Username == "" {
			errs = append(errs, errors2.APIError{
				Message:    "username is required",
				HTTPStatus: http.StatusBadRequest,
			})
		}
		if dataToChange.Password == "" {
			errs = append(errs, errors2.APIError{
				Message:    "password is required",
				HTTPStatus: http.StatusBadRequest,
			})
		}
		if as.TwoFAOn && dataToChange.TwoFASendTo == "" {
			errs = append(errs, errors2.APIError{
				Message:    "two_fa_send_to is required",
				HTTPStatus: http.StatusBadRequest,
			})
		}
	} else {
		if (dataToChange.Username == "" || dataToChange.Username == usernameToFind) &&
			dataToChange.Password == "" && dataToChange.Groups == nil && (!as.TwoFAOn || dataToChange.TwoFASendTo == "") && dataToChange.Token == nil {
			errs = append(errs, errors2.APIError{
				Message:    "nothing to change",
				HTTPStatus: http.StatusBadRequest,
			})
		}
	}

	if dataToChange.TwoFASendTo != "" && as.DeliverySrv != nil {
		// TODO: use proper ctx when it will be propagated
		err := as.DeliverySrv.ValidateReceiver(context.Background(), dataToChange.TwoFASendTo)
		if err != nil {
			errs = append(errs, errors2.APIError{
				Err:        fmt.Errorf("invalid two_fa_send_to: %v", err),
				HTTPStatus: http.StatusBadRequest,
			})
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}

func (as *APIService) addUser(dataToChange *User) error {
	existingUser, err := as.Provider.GetByUsername(dataToChange.Username)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return errors2.APIError{
			Message:    "Another user with this username already exists",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	err = as.Provider.Add(dataToChange)
	if err != nil {
		return err
	}

	return nil
}

// todo make concurrent save
func (as *APIService) updateUser(dataToChange *User, usernameToFind string) error {
	existingUser, err := as.Provider.GetByUsername(usernameToFind)
	if err != nil {
		return err
	}

	if existingUser == nil {
		return errors2.APIError{
			Message:    fmt.Sprintf("cannot find user by username '%s'", usernameToFind),
			HTTPStatus: http.StatusNotFound,
		}
	}

	if dataToChange.Username != "" && dataToChange.Username != usernameToFind {
		existingUser, err := as.Provider.GetByUsername(dataToChange.Username)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return errors2.APIError{
				Message:    "Another user with this username already exists",
				HTTPStatus: http.StatusBadRequest,
			}
		}
	}

	err = as.Provider.Update(dataToChange, usernameToFind)
	if err != nil {
		return err
	}
	return nil
}

func (as *APIService) Delete(usernameToDelete string) error {
	user, err := as.Provider.GetByUsername(usernameToDelete)
	if err != nil {
		return err
	}

	if user == nil {
		return errors2.APIError{
			Message:    fmt.Sprintf("cannot find user by username '%s'", usernameToDelete),
			HTTPStatus: http.StatusNotFound,
		}
	}

	return as.Provider.Delete(usernameToDelete)
}
