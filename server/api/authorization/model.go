package command

import (
	"time"
)

const DefaultTimeoutSec = 60

type ApiTokens struct {
	ID        string     `json:"id,omitempty" db:"id"`
	Username  string     `json:"name,omitempty" db:"username"`
	Prefix    string     `json:"name,omitempty" db:"prefix"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
	ExpiresAt *time.Time `json:"updated_at,omitempty" db:"expires_at"`
	Scope     string     `json:"cmd,omitempty" db:"scope"`
	Token     string     `json:"cmd,omitempty" db:"token"`
}
