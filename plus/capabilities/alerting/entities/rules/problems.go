package rules

import (
	"time"
)

type ProblemID string

type ProblemState string

const (
	ProblemActive   ProblemState = "ACTIVE"
	ProblemResolved ProblemState = "RESOLVED"
)

type Problem struct {
	ID        ProblemID    `json:"problem_id"`
	RuleSetID RuleSetID    `json:"rule_set_id"`
	RuleID    RuleID       `json:"rule_id"`
	ClientID  string       `json:"client_id"`
	Actions   ActionList   `json:"actions"`
	State     ProblemState `json:"state"`

	CreatedAt  time.Time `json:"created_at"`
	ResolvedAt time.Time `json:"resolved_at"`

	CUID string `json:"client_update_id"`
	MUID string `json:"measure_update_id"`
}

func (p *Problem) Clone() (clonedProblem Problem) {
	clonedProblem = *p
	clonedProblem.Actions = p.Actions.Clone()
	return clonedProblem
}

type Problems []Problem

type ProblemUpdateRequest struct {
	State ProblemState `json:"state"`
}
