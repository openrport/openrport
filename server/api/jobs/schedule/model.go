package schedule

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api/jobs"
)

const (
	TypeCommand = "command"
	TypeScript  = "script"
)

type Schedule struct {
	Base
	Details
	LastExecution *Execution `json:"last_execution"`
}

func (s Schedule) ToDB() DBSchedule {
	return DBSchedule{
		Base:    s.Base,
		Details: s.Details,
	}
}

// DBSchedule is used for saving to database and has details in one json db column
type DBSchedule struct {
	Base
	Details Details `db:"details"`
	Execution
}

func (dbs DBSchedule) ToSchedule() *Schedule {
	return &Schedule{
		Base:          dbs.Base,
		Details:       dbs.Details,
		LastExecution: dbs.Execution.ToLastExecution(),
	}
}

type Base struct {
	ID        string    `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	CreatedBy string    `json:"created_by" db:"created_by"`
	Name      string    `json:"name" db:"name"`
	Schedule  string    `json:"schedule" db:"schedule"`
	Type      string    `json:"type" db:"type"`
}

type Details struct {
	ClientIDs           []string `json:"client_ids" db:"-"`
	GroupIDs            []string `json:"group_ids" db:"-"`
	Command             string   `json:"command,omitempty" db:"-"`
	Script              string   `json:"script,omitempty" db:"-"`
	Interpreter         string   `json:"interpreter" db:"-"`
	Cwd                 string   `json:"cwd" db:"-"`
	IsSudo              bool     `json:"is_sudo" db:"-"`
	TimeoutSec          int      `json:"timeout_sec" db:"-"`
	ExecuteConcurrently bool     `json:"execute_concurrently" db:"-"`
	AbortOnError        *bool    `json:"abort_on_error" db:"-"`
	Overlaps            bool     `json:"overlaps" db:"-"`
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

// All fields must be pointers, because when there's no execution yet the values will be nil
type Execution struct {
	StartedAt    *time.Time `db:"last_started_at" json:"started_at"`
	ClientCount  *int       `db:"last_client_count" json:"client_count"`
	SuccessCount *int       `db:"last_success_count" json:"success_count"`
	Status       *string    `db:"last_status" json:"status"`

	Details *jobs.JobDetails `db:"last_details" json:"-"`
	Summary *string          `json:"summary"`
}

func (e *Execution) ToLastExecution() *Execution {
	if e.StartedAt == nil {
		return nil
	}

	if e.ClientCount != nil {
		if *e.ClientCount == 1 {
			if e.Details != nil && e.Details.Result != nil {
				e.Summary = &e.Details.Result.Summary
			}
		} else {
			e.Status = nil

		}
	}
	return e
}
