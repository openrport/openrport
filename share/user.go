package chshare

import (
	"strings"
)

func ParseAuth(auth string) (string, string) {
	if strings.Contains(auth, ":") {
		pair := strings.SplitN(auth, ":", 2)
		return pair[0], pair[1]
	}
	return "", ""
}

type User struct {
	Name string
	Pass string
}
