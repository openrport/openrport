package authorization

import (
	"errors"
	"strings"
	"time"
)

type APIToken struct {
	Username  string     `json:"username,omitempty" db:"username"`
	Prefix    string     `json:"prefix" db:"prefix"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	Scope     string     `json:"scope,omitempty" db:"scope"`
	Token     string     `json:"token,omitempty" db:"token"`
}

func Extract(prefixedpwd string) (string, string, error) {
	i := strings.Index(prefixedpwd, "_")
	if i < 0 {
		return "", "", errors.New("token should be in the format 'prefix_token'")
	}
	if i != 8 {
		return "", "", errors.New("invalid token")
	}
	prefix := prefixedpwd[0:i]
	token := prefixedpwd[i+1:]
	return prefix, token, nil
}

// TODO: hardcode the scopes in one place only
func IsValidScope(scope string) bool {
	switch scope {
	case
		"read",
		"read+write",
		"clients-auth":
		return true
	}
	return false
}
