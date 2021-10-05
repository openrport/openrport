package command

import "time"

// To support sparse fieldsets, the fields that can have zero value,
// use pointers so they're omitted only when they're nil not when they're zero value
type Command struct {
	ID        string     `json:"id,omitempty" db:"id"`
	Name      string     `json:"name,omitempty" db:"name"`
	CreatedBy string     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedBy string     `json:"updated_by,omitempty" db:"updated_by"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	Cmd       string     `json:"cmd,omitempty" db:"cmd"`
}

type InputCommand struct {
	Name string `json:"name" db:"name"`
	Cmd  string `json:"cmd" db:"script"`
}
