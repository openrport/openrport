package authorization

import (
	"time"
)

const DefaultTimeoutSec = 60

type APIToken struct {
	Username  string     `json:"username" db:"username"`
	Prefix    string     `json:"prefix" db:"prefix"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	Scope     string     `json:"scope,omitempty" db:"scope"`
	Token     string     `json:"token,omitempty" db:"token"`
}

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
