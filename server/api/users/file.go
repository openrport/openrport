package users

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/realvnc-labs/rport/share/logger"
)

const HtpasswdBcryptPrefix = "$2y$"
const HtpasswdBcryptAltPrefix = "$2a$"

type FileManager struct {
	*logger.Logger

	FileName       string
	FileAccessLock sync.Mutex
}

func NewFileManager(logger *logger.Logger, fileName string) *FileManager {
	return &FileManager{
		Logger:   logger,
		FileName: fileName,
	}
}

// ReadUsersFromFile returns users from a given file.
func (fm *FileManager) ReadUsersFromFile() ([]*User, error) {
	fm.FileAccessLock.Lock()
	defer fm.FileAccessLock.Unlock()
	fm.Logger.Infof("Start to get API users from file.")

	file, err := os.Open(fm.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open users file: %v", err)
	}
	fm.Logger.Infof("API users file %s opened. Parsing...", fm.FileName)
	defer file.Close()

	users, err := parseUsers(file)
	if err != nil {
		return users, err
	}
	fm.Logger.Infof("API users file %s is parsed successfully", fm.FileName)

	return users, nil
}

// SaveUsersToFile writes users to a file in json format
func (fm *FileManager) SaveUsersToFile(usrs []*User) error {
	fm.FileAccessLock.Lock()
	defer fm.FileAccessLock.Unlock()

	file, err := json.MarshalIndent(usrs, "", " ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fm.FileName, file, 0600)
}

func parseUsers(r io.Reader) ([]*User, error) {
	decoder := json.NewDecoder(r)
	// read array open bracket
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to parse users data: %v", err)
	}

	var users []*User
	usernames := make(map[string]bool)
	for decoder.More() {
		var user User
		if err := decoder.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to parse user: %v", err)
		}

		u := strings.TrimSpace(user.Username)
		if u == "" {
			return nil, errors.New("username can not be empty")
		}
		user.Username = u

		p := strings.TrimSpace(user.Password)
		if p == "" {
			return nil, errors.New("password can not be empty")
		}
		if !strings.HasPrefix(p, HtpasswdBcryptPrefix) {
			return nil, fmt.Errorf("username %q: require passwords to be bcrypt hashed and to be compatible with \"htpasswd -bnBC 10 \"\" <password> | tr -d ':'\" ", user.Username)
		}
		user.Password = p

		if usernames[user.Username] {
			return nil, fmt.Errorf("non unique username: %q", user.Username)
		}

		usernames[user.Username] = true
		users = append(users, &user)
	}

	return users, nil
}
