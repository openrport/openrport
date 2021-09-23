package script

import "time"

// To support sparse fieldsets, the fields that can have zero value,
// use pointers so they're omitted only when they're nil not when they're zero value
type Script struct {
	ID          string     `json:"id,omitempty" db:"id"`
	Name        string     `json:"name,omitempty" db:"name"`
	CreatedBy   string     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
	Interpreter *string    `json:"interpreter,omitempty" db:"interpreter"`
	IsSudo      *bool      `json:"is_sudo,omitempty" db:"is_sudo"`
	Cwd         *string    `json:"cwd,omitempty" db:"cwd"`
	Script      string     `json:"script,omitempty" db:"script"`
}

type InputScript struct {
	Name        string `json:"name" db:"name"`
	Interpreter string `json:"interpreter" db:"interpreter"`
	IsSudo      bool   `json:"is_sudo" db:"is_sudo"`
	Cwd         string `json:"cwd" db:"cwd"`
	Script      string `json:"script" db:"script"`
}
