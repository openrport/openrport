package users

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/share/enums"
)

type DatabaseProvider interface {
	GetAll() ([]*User, error)
	GetByUsername(username string) (*User, error)
	Add(usr *User) error
	Update(usr *User, usernameToUpdate string) error
	Delete(usernameToDelete string) error
}

type FileProvider interface {
	ReadUsersFromFile() ([]*User, error)
	SaveUsersToFile(users []*User) error
}

type APIService struct {
	ProviderType enums.ProviderSource
	FileProvider FileProvider
	DeliverySrv  message.Service
	DB           DatabaseProvider
	TwoFAOn      bool
}

func (as *APIService) GetAll() ([]*User, error) {
	switch as.ProviderType {
	case enums.ProviderSourceFile:
		authUsers, err := as.FileProvider.ReadUsersFromFile()
		if err != nil {
			return nil, err
		}
		return authUsers, nil
	case enums.ProviderSourceDB:
		usrs, err := as.DB.GetAll()
		if err != nil {
			return nil, err
		}
		return usrs, nil
	}

	return nil, fmt.Errorf("unsupported user data provider type: %s", as.ProviderType)
}

func (as *APIService) Change(usr *User, username string) error {
	if usr.Password != "" {
		passHash, err := bcrypt.GenerateFromPassword([]byte(usr.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		usr.Password = strings.Replace(string(passHash), htpasswdBcryptAltPrefix, htpasswdBcryptPrefix, 1)
	}

	err := as.validate(usr, username)
	if err != nil {
		return err
	}

	if as.ProviderType == enums.ProviderSourceFile {
		return as.changeUserInFile(usr, username)
	}

	if as.ProviderType == enums.ProviderSourceDB {
		return as.changeUserInDB(usr, username)
	}

	return fmt.Errorf("unsupported user data provider type: %s", as.ProviderType)
}

func (as *APIService) validate(dataToChange *User, usernameToFind string) error {
	errs := errors2.APIErrors{}

	if usernameToFind == "" {
		if dataToChange.Username == "" {
			errs = append(errs, errors2.APIError{
				Message: "username is required",
				Code:    http.StatusBadRequest,
			})
		}
		if dataToChange.Password == "" {
			errs = append(errs, errors2.APIError{
				Message: "password is required",
				Code:    http.StatusBadRequest,
			})
		}
		if as.TwoFAOn && dataToChange.TwoFASendTo == "" {
			errs = append(errs, errors2.APIError{
				Message: "two_fa_send_to is required",
				Code:    http.StatusBadRequest,
			})
		}
	} else {
		if (dataToChange.Username == "" || dataToChange.Username == usernameToFind) &&
			dataToChange.Password == "" && dataToChange.Groups == nil && dataToChange.TwoFASendTo == "" {
			errs = append(errs, errors2.APIError{
				Message: "nothing to change",
				Code:    http.StatusBadRequest,
			})
		}
	}

	if dataToChange.TwoFASendTo != "" && as.DeliverySrv != nil {
		err := as.DeliverySrv.ValidateReceiver(dataToChange.TwoFASendTo)
		if err != nil {
			errs = append(errs, errors2.APIError{
				Err:  fmt.Errorf("invalid two_fa_send_to: %v", err),
				Code: http.StatusBadRequest,
			})
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}

func (as *APIService) addUserToDB(dataToChange *User) error {
	existingUser, err := as.DB.GetByUsername(dataToChange.Username)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return errors2.APIError{
			Message: "Another user with this username already exists",
			Code:    http.StatusBadRequest,
		}
	}

	err = as.DB.Add(dataToChange)
	if err != nil {
		return err
	}

	return nil
}

// todo make concurrent save
func (as *APIService) updateUserInDB(dataToChange *User, usernameToFind string) error {
	existingUser, err := as.DB.GetByUsername(usernameToFind)
	if err != nil {
		return err
	}

	if existingUser == nil {
		return errors2.APIError{
			Message: fmt.Sprintf("cannot find user by username '%s'", usernameToFind),
			Code:    http.StatusNotFound,
		}
	}

	if dataToChange.Username != "" && dataToChange.Username != usernameToFind {
		existingUser, err := as.DB.GetByUsername(dataToChange.Username)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return errors2.APIError{
				Message: "Another user with this username already exists",
				Code:    http.StatusBadRequest,
			}
		}
	}

	err = as.DB.Update(dataToChange, usernameToFind)
	if err != nil {
		return err
	}
	return nil
}

func (as *APIService) changeUserInDB(dataToChange *User, usernameToFind string) error {
	if usernameToFind == "" {
		return as.addUserToDB(dataToChange)
	}
	return as.updateUserInDB(dataToChange, usernameToFind)
}

func (as *APIService) addUserToFile(dataToChange *User) error {
	users, err := as.FileProvider.ReadUsersFromFile()
	if err != nil {
		return err
	}

	for i := range users {
		if users[i].Username == dataToChange.Username {
			return errors2.APIError{
				Message: "Another user with this username already exists",
				Code:    http.StatusBadRequest,
			}
		}
	}

	users = append(users, dataToChange)
	err = as.FileProvider.SaveUsersToFile(users)
	if err != nil {
		return err
	}
	return nil
}

func (as *APIService) updateUserInFile(dataToChange *User, usernameToFind string) error {
	users, err := as.FileProvider.ReadUsersFromFile()
	if err != nil {
		return err
	}

	userFound := -1
	for i := range users {
		if users[i].Username == usernameToFind {
			userFound = i
		}
		if dataToChange.Username != "" && users[i].Username == dataToChange.Username && dataToChange.Username != usernameToFind {
			return errors2.APIError{
				Message: "Another user with this username already exists",
				Code:    http.StatusBadRequest,
			}
		}
	}

	if userFound < 0 {
		return errors2.APIError{
			Message: fmt.Sprintf("cannot find user by username '%s'", usernameToFind),
			Code:    http.StatusNotFound,
		}
	}

	if dataToChange.Password != "" {
		users[userFound].Password = dataToChange.Password
	}
	if dataToChange.Groups != nil {
		users[userFound].Groups = dataToChange.Groups
	}
	if dataToChange.Username != "" {
		users[userFound].Username = dataToChange.Username
	}

	err = as.FileProvider.SaveUsersToFile(users)
	if err != nil {
		return err
	}

	return nil
}

func (as *APIService) changeUserInFile(dataToChange *User, usernameToFind string) error {
	if usernameToFind != "" {
		return as.updateUserInFile(dataToChange, usernameToFind)
	}

	return as.addUserToFile(dataToChange)
}

func (as *APIService) Delete(usernameToDelete string) error {
	if as.ProviderType == enums.ProviderSourceFile {
		return as.deleteUserFromFile(usernameToDelete)
	}

	if as.ProviderType == enums.ProviderSourceDB {
		return as.deleteUserFromDB(usernameToDelete)
	}

	return fmt.Errorf("unsupported user data provider type: %s", as.ProviderType)
}

func (as *APIService) deleteUserFromDB(usernameToDelete string) error {
	user, err := as.DB.GetByUsername(usernameToDelete)
	if err != nil {
		return err
	}

	if user == nil {
		return errors2.APIError{
			Message: fmt.Sprintf("cannot find user by username '%s'", usernameToDelete),
			Code:    http.StatusNotFound,
		}
	}

	return as.DB.Delete(usernameToDelete)
}

func (as *APIService) deleteUserFromFile(usernameToDelete string) error {
	usersFromFile, err := as.FileProvider.ReadUsersFromFile()
	if err != nil {
		return err
	}
	foundIndex := -1
	for i := range usersFromFile {
		if usersFromFile[i].Username == usernameToDelete {
			foundIndex = i
			break
		}
	}

	if foundIndex < 0 {
		return errors2.APIError{
			Message: fmt.Sprintf("cannot find user by username '%s'", usernameToDelete),
			Code:    http.StatusNotFound,
		}
	}

	usersToWriteToFile := append(usersFromFile[:foundIndex], usersFromFile[foundIndex+1:]...)
	err = as.FileProvider.SaveUsersToFile(usersToWriteToFile)
	if err != nil {
		return err
	}
	return nil
}
