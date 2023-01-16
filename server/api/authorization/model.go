package authorization

import (
	"errors"

	"strings"
	"time"
)

type APIToken struct {
	Username  string        `json:"username,omitempty" db:"username"`
	Prefix    string        `json:"prefix" db:"prefix"`
	CreatedAt *time.Time    `json:"created_at,omitempty" db:"created_at"`
	ExpiresAt *time.Time    `json:"expires_at,omitempty" db:"expires_at"`
	Scope     APITokenScope `json:"scope,omitempty" db:"scope"`
	Token     string        `json:"token,omitempty" db:"token"`
}

const APITokenPrefixLength = 8

type APITokenScope string

const (
	APITokenRead        APITokenScope = "read"
	APITokenReadWrite   APITokenScope = "read+write"
	APITokenClientsAuth APITokenScope = "clients-auth"
)

func Extract(prefixedpwd string) (string, string, error) {
	i := strings.Index(prefixedpwd, "_")
	if i < 0 {
		return "", "", errors.New("token should be in the format 'prefix_token'")
	}
	if i != APITokenPrefixLength {
		return "", "", errors.New("invalid token")
	}
	parts := strings.SplitN(prefixedpwd, "_", i)
	prefix := parts[0]
	token := parts[1]
	return prefix, token, nil
}

func IsValidScope(scope APITokenScope) bool {
	switch scope {
	case
		APITokenRead,
		APITokenReadWrite,
		APITokenClientsAuth:
		return true
	}
	return false
}
