package rules

import (
	"time"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/actions"
)

type ActionState string

const (
	ActionActive    ActionState = "ACTIVE"
	ActionNotActive ActionState = "NOT_ACTIVE"
	ActionResolved  ActionState = "RESOLVED"
)

type RuleActionState struct {
	RuleSetID RuleSetID `json:"id"`
	RuleID    RuleID    `json:"rule_id"`
	ClientID  string    `json:"client_id"`

	ActType     actions.AT  `json:"act_type"`
	ActionState ActionState `json:"act_state"`

	CreatedAt  time.Time `json:"created_at"`
	ResolvedAt time.Time `json:"resolved_at"`

	CUID string `json:"client_update_id"`
	MUID string `json:"measure_update_id"`
}
