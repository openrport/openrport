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

type UserVars map[string]any

type RuleSet struct {
	RuleSetID RuleSetID `json:"id,omitempty"`
	Vars      UserVars  `json:"vars,omitempty"`
	Rules     []Rule    `json:"rules"`
}

type State string

const (
	StateUnknown State = "Unknown"
	NotFiring    State = "Not Firing"
	Firing       State = "Firing"
)

type Rule struct {
	ID       RuleID            `json:"id"`
	Severity severity.Severity `json:"severity"`
	Ex       string            `json:"expr"`
	Actions  ActionList        `json:"actions"`
}

func (r *Rule) Clone() (clonedRule Rule) {
	clonedRule = *r
	clonedRule.Actions = r.Actions.Clone()
	return clonedRule
}

type ActionList []Action

func (al ActionList) Clone() (clonedAL ActionList) {
	for _, action := range al {
		clonedAL = append(clonedAL, action.Clone())
	}

	return clonedAL
}

type NotifyList []templates.TemplateID

type LogMessage string

type IgnoreList []IgnoreSpec

type IgnoreSpec string

type Action struct {
	*NotifyList `json:"notify,omitempty"`
	*IgnoreList `json:"ignore,omitempty"`
	LogMessage  `json:"log,omitempty"`
}

func (at *Action) GetActType() (actType actions.ActionType) {
	if at.NotifyList != nil {
		return actions.NotifyType
	}
	if at.IgnoreList != nil {
		return actions.IgnoreType
	}
	if at.LogMessage != "" {
		return actions.LogType
	}
	return actions.UnknownType
}

func (at *Action) Clone() (clonedAct Action) {
	clonedAct = Action{}
	if at.NotifyList != nil {
		notifyList := make(NotifyList, len(*at.NotifyList))
		copy(notifyList, *at.NotifyList)
		clonedAct.NotifyList = &notifyList
	}
	if at.IgnoreList != nil {
		ignoreList := make(IgnoreList, len(*at.NotifyList))
		copy(ignoreList, *at.IgnoreList)
		clonedAct.IgnoreList = &ignoreList
	}
	clonedAct.LogMessage = at.LogMessage
	return clonedAct
}
