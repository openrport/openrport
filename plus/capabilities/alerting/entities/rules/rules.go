package rules

import (
	"errors"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/actions"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/templates"
)

var (
	ErrRuleSetNotFound = errors.New("rule set not found")
)

type RuleSetID string
type RuleID string

type UserParams map[string]any

type RuleSet struct {
	RuleSetID RuleSetID  `mapstructure:"id" json:"id"`
	Params    UserParams `mapstructure:"params" json:"params"`
	Rules     []Rule     `mapstructure:"rules" json:"rules"`
}

type Rule struct {
	ID      RuleID     `mapstructure:"id" json:"id"`
	Ex      string     `mapstructure:"expr" json:"expr"`
	Actions ActionList `mapstructure:"action" json:"actions"`
}

type ActionList []Action

type Action struct {
	ActType       actions.AT `mapstructure:"type" json:"act_type"`
	*NotifyAction `mapstructure:",squash" json:"notify_action,omitempty"`
	*IgnoreAction `mapstructure:",squash" json:"ignore_action,omitempty"`
	*LogAction    `mapstructure:",squash" json:"log_action,omitempty"`
}

type NotifyAction struct {
	TemplateIDs []templates.TemplateID `mapstructure:"body" json:"template_ids"`
}

type LogAction struct {
	LogStr string `mapstructure:"log_str" json:"log_str"`
}

type IgnoreAction struct {
	Match string `mapstructure:"match" json:"match"`
}
