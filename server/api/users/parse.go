package users

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const htpasswdBcryptPrefix = "$2y$"

// GetUsersFromFile returns users from a given file.
func GetUsersFromFile(fileName string) ([]*User, error) {
	log.Println("Start to get users from file.")

	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open users file: %v", err)
	}
	log.Println("Users file opened. Parsing...")
	defer file.Close()

	return parseUsers(file)
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
		if !strings.HasPrefix(p, htpasswdBcryptPrefix) {
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
