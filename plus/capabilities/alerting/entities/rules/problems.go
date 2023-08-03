package rules

import (
	"time"

	"github.com/realvnc-labs/rport/share/types"
)

type ProblemID string

type Problem struct {
	ID         ProblemID  `json:"problem_id"`
	RuleID     RuleID     `json:"rule_id"`
	ClientID   string     `json:"client_id"`
	ClientName string     `json:"client_name"`
	Actions    ActionList `json:"actions"`
	Active     bool       `json:"active"`

	CreatedAt  time.Time      `json:"created_at"`
	ResolvedAt types.TimeJSON `json:"resolved_at"`

	CUID string `json:"client_update_id"`
	MUID string `json:"measurement_update_id"`
}

func (p *Problem) Clone() (clonedProblem Problem) {
	clonedProblem = *p
	clonedProblem.Actions = p.Actions.Clone()
	return clonedProblem
}

type Problems []Problem

type ProblemUpdateRequest struct {
	Active bool `json:"active"`
}
