package chshare

import (
	"os/user"
	"strings"
)

func ParseAuth(auth string) (string, string) {
	if strings.Contains(auth, ":") {
		pair := strings.SplitN(auth, ":", 2)
		return pair[0], pair[1]
	}
	return "", ""
}

func GetCurrentUserAndGroup() (*user.User, *user.Group, error) {
	curUser, err := user.Current()
	if err != nil {
		return nil, nil, err
	}

	gr, err := user.LookupGroupId(curUser.Gid)
	if err != nil {
		return nil, nil, err
	}

	return curUser, gr, nil
}
