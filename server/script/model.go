package script

import "time"

type Script struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	CreatedBy   string    `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	Interpreter string    `json:"interpreter" db:"interpreter"`
	IsSudo      bool      `json:"is_sudo" db:"is_sudo"`
	Cwd         string    `json:"cwd" db:"cwd"`
	Script      string    `json:"script" db:"script"`
}

type InputScript struct {
	Name        string `json:"name" db:"name"`
	Interpreter string `json:"interpreter" db:"interpreter"`
	IsSudo      bool   `json:"is_sudo" db:"is_sudo"`
	Cwd         string `json:"cwd" db:"cwd"`
	Script      string `json:"script" db:"script"`
}
