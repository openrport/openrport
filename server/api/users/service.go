package users

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	zxcvbn "github.com/trustelem/zxcvbn"
	"golang.org/x/crypto/bcrypt"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/share/enums"
)

type Provider interface {
	Type() enums.ProviderSource
	SupportsGroupPermissions() bool
	GetAll() ([]*User, error)
	ListGroups() ([]Group, error)
	GetGroup(string) (Group, error)
	UpdateGroup(string, Group) error
	DeleteGroup(string) error
	GetByUsername(username string) (*User, error)
	Add(usr *User) error
	Update(usr *User, usernameToUpdate string) error
	Delete(usernameToDelete string) error
}

type APIService struct {
	DeliverySrv            message.Service
	Provider               Provider
	TwoFAOn                bool
	TotPOn                 bool
	PasswordMinLength      int
	PasswordZxcvbnMinscore int
}

func NewAPIService(provider Provider, twoFAOn bool, passwordMinLength int, PasswordZxcvbnMinscore int) *APIService {
	return &APIService{
		Provider:               provider,
		TwoFAOn:                twoFAOn,
		PasswordMinLength:      passwordMinLength,
		PasswordZxcvbnMinscore: PasswordZxcvbnMinscore,
	}
}

func (as APIService) SupportsGroupPermissions() bool {
	return as.Provider.SupportsGroupPermissions()
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

func (as *APIService) ListGroups() ([]Group, error) {
	return as.Provider.ListGroups()
}

func (as *APIService) GetGroup(name string) (Group, error) {
	return as.Provider.GetGroup(name)
}

func (as *APIService) UpdateGroup(name string, g Group) (Group, error) {
	err := as.Provider.UpdateGroup(name, g)
	if err != nil {
		return Group{}, err
	}
	return as.Provider.GetGroup(name)
}

func (as *APIService) DeleteGroup(name string) error {
	return as.Provider.DeleteGroup(name)
}

func (as *APIService) CheckPermission(user *User, permission string) error {
	for _, groupName := range user.Groups {
		group, err := as.Provider.GetGroup(groupName)
		if err != nil {
			return err
		}
		if group.Permissions.Has(permission) {
			return nil
		}
	}
	return errors2.APIError{
		Message:    fmt.Sprintf("user does not have %q permission", permission),
		HTTPStatus: http.StatusForbidden,
	}
}

func (as *APIService) GetEffectiveUserPermissions(user *User) (map[string]bool, error) {
	if !as.SupportsGroupPermissions() {
		return NewPermissions(AllPermissions...).All(), nil
	}
	permissions := NewPermissions().All()

	for _, groupName := range user.Groups {
		group, err := as.Provider.GetGroup(groupName)
		if err != nil {
			return permissions, err
		}
		for _, permission := range AllPermissions {
			if group.Permissions.Has(permission) && !permissions[permission] {
				permissions[permission] = true
			}
		}
	}
	return permissions, nil
}

func (as *APIService) ExistGroups(groups []string) error {
	existingGroups, err := as.ListGroups()
	if err != nil {
		return err
	}

	var groupsNotFound []string
	for _, group := range groups {
		found := false
		for _, existing := range existingGroups {
			if existing.Name == group {
				found = true
				break
			}
		}
		if !found {
			groupsNotFound = append(groupsNotFound, group)
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
	err := as.validate(usr, username) //validate does it all validation
	if err != nil {
		return err
	}
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
	if username != "" {
		return as.updateUser(usr, username)
	}
	return as.addUser(usr)
}

func (as *APIService) validate(dataToChange *User, usernameToFind string) error {
	//    2685 password validation happens here, and it should be good for any level
	fmt.Printf("**** validate User: %#v\n", dataToChange)
	fmt.Printf("**** usernameToFind: %#v\n", usernameToFind)
	fmt.Printf("**** dataToChange.Username: %#v\n", dataToChange.Username)
	errs := errors2.APIErrors{}

	// TODO: need to move the password encoding after the validation!
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
			dataToChange.Password == "" &&
			dataToChange.Groups == nil &&
			(!as.TwoFAOn || dataToChange.TwoFASendTo == "") &&
			dataToChange.Token == nil &&
			dataToChange.TotP == "" {
			errs = append(errs, errors2.APIError{
				Message:    "nothing to change",
				HTTPStatus: http.StatusBadRequest,
			})
		}
	}

	//    password validation
	if dataToChange.Password != "" { // curl -Ss -X PUT http://localhost:3000/api/v1/users/eddy -u Admin:ciccio -H "content-type:application/json" --data-raw '{"password": ""}'
		if len(dataToChange.Password) < as.PasswordMinLength { // TODO: 14 needs to be in config
			errs = append(errs, errors2.APIError{
				Message:    fmt.Sprintf("password must be at least %v characters", as.PasswordMinLength),
				HTTPStatus: http.StatusBadRequest,
			})
		}
		fmt.Printf("**** as.PasswordZxcvbnMinscore: %#v\n", as.PasswordZxcvbnMinscore)
		if as.PasswordZxcvbnMinscore >= 0 { // -1 means no zxcvbn
			score := zxcvbn.PasswordStrength(string(dataToChange.Password), nil)
			fmt.Printf("**** score: %#v\n", score)

			if score.Score < as.PasswordZxcvbnMinscore { // TODO: move 5 into a new config parm or use 0 NO zxcvbn, 1 ... the value
				errs = append(errs, errors2.APIError{
					Message:    fmt.Sprintf("zxcvbn score is %v, must be at least %v", score.Score, as.PasswordZxcvbnMinscore),
					HTTPStatus: http.StatusBadRequest,
				})
			}
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
