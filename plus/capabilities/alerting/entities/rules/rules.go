package rules

import (
	"errors"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/actions"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/severity"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/templates"
)

const (
	DefaultRuleSetID RuleSetID = "default"
)

var (
	ErrRuleSetValidationFailed = errors.New("rule set validation failed")

	ErrMissingRulesMsg                 = "there must be at least 1 rule in a rule set"
	ErrMissingRuleIDMsg                = "rule id cannot be empty"
	ErrMissingNotificationTemplatesMsg = "missing notification templates"
	ErrMissingIgnoreSpecsMsg           = "missing ignore specs"
	ErrActionMissingContentMsg         = "missing action content"
	ErrMissingExprMsg                  = "expression cannot be empty"
	ErrTemplateNotFoundMsg             = "template not found"
	ErrFailedToCompileMsg              = "failed to compile rule"
)

type AlertStatus string

const (
	Alerting AlertStatus = "ALERTING"
	Resolved AlertStatus = "RESOLVED"
)

type RuleSetID string
type RuleID string

type UserParams map[string]any

type RuleSet struct {
	RuleSetID RuleSetID  `mapstructure:"id" json:"id,omitempty"`
	Params    UserParams `mapstructure:"params" json:"params,omitempty"`
	Rules     []Rule     `mapstructure:"rules" json:"rules"`
}

type State string

const (
	StateUnknown State = "UNKNOWN"
	NotFiring    State = "NOT_FIRING"
	Firing       State = "FIRING"
)

type Rule struct {
	ID       RuleID            `mapstructure:"id" json:"id"`
	Severity severity.Severity `mapstructure:"severity" json:"severity"`
	Ex       string            `mapstructure:"expr" json:"expr"`
	Actions  ActionList        `mapstructure:"action" json:"actions"`
}

type ActionList []Action

type Action struct {
	*NotifyList `mapstructure:",squash" json:"notify,omitempty"`
	*IgnoreList `mapstructure:",squash" json:"ignore,omitempty"`
	LogMessage  `mapstructure:",squash" json:"log,omitempty"`
}

func (at *Action) GetActType() (actType actions.AT) {
	if at.NotifyList != nil {
		return actions.NotifyActionType
	}
	if at.IgnoreList != nil {
		return actions.IgnoreActionType
	}
	if at.LogMessage != "" {
		return actions.LogActionType
	}
	return actions.UnknownActionType
}

type NotifyList []templates.TemplateID

type LogMessage string

type IgnoreList []IgnoreSpec

type IgnoreSpec string
