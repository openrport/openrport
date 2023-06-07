package rules

import (
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/actions"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/notifications"
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
	Type            notifications.NotificationType `mapstructure:"type" json:"type"`
	Recipients      notifications.RecipientList    `mapstructure:"recipients" json:"recipients"`
	SubjectTemplate string                         `mapstructure:"subject" json:"subject"`
	BodyTemplate    string                         `mapstructure:"body" json:"body"`
}

type LogAction struct {
	LogStr string `mapstructure:"log_str" json:"log_str"`
}

type IgnoreAction struct {
	Match string `mapstructure:"match" json:"match"`
}
