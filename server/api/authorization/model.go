package command

import (
	"time"
)

const DefaultTimeoutSec = 60

type ApiTokens struct {
	ID        string     `json:"id,omitempty" db:"id"`
	Username  string     `json:"username,omitempty" db:"username"`
	Prefix    string     `json:"prefix,omitempty" db:"prefix"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	Scope     string     `json:"scope,omitempty" db:"scope"`
	Token     string     `json:"token,omitempty" db:"token"`
}
