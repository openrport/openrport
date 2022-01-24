package schedule

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	TypeCommand = "command"
	TypeScript  = "script"
)

type Schedule struct {
	ID        string    `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	CreatedBy string    `json:"created_by" db:"created_by"`
	Name      string    `json:"name" db:"name"`
	Schedule  string    `json:"schedule" db:"schedule"`
	Type      string    `json:"type" db:"type"`
	Details   Details   `json:"details" db:"details"`
}

type Details struct {
	ClientIDs           []string `json:"client_ids"`
	GroupIDs            []string `json:"group_ids"`
	Command             string   `json:"command,omitempty"`
	Script              string   `json:"script,omitempty"`
	Interpreter         string   `json:"interpreter"`
	Cwd                 string   `json:"cwd"`
	IsSudo              bool     `json:"is_sudo"`
	TimeoutSec          int      `json:"timeout_sec"`
	ExecuteConcurrently bool     `json:"execute_concurrently"`
	AbortOnError        *bool    `json:"abort_on_error"`
	Overlaps            bool     `json:"overlaps"`
}

func (d *Details) Scan(value interface{}) error {
	if d == nil {
		return errors.New("'details' cannot be nil")
	}
	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected to have string, got %T", value)
	}
	err := json.Unmarshal([]byte(valueStr), d)
	if err != nil {
		return fmt.Errorf("failed to decode 'details' field: %v", err)
	}
	return nil
}

func (d Details) Value() (driver.Value, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("failed to encode 'details' field: %v", err)
	}
	return string(b), nil
}

type Execution struct {
	ScheduleID string
	CreatedAt  time.Time
	Error      string
	MultiJobID string
}
