package alertingcap

import (
	"context"

	"github.com/dgraph-io/badger/v4"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/clientupdates"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/measures"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rules"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/templates"
)

type CapabilityEx interface {
	Init(db *badger.DB) (err error)

	GetService() (as Service)
}

type Config struct {
	AlertsLogPath string `mapstructure:"alert_log_path"`
}

type Service interface {
	Run(ctx context.Context)
	Stop() (err error)

	LoadRuleSet(ruleSetID rules.RuleSetID) (rs *rules.RuleSet, err error)
	SaveRuleSet(rs *rules.RuleSet) (err error)
	DeleteRuleSet(ruleSetID rules.RuleSetID) (err error)

	GetAllTemplates() (templateList templates.TemplateList, err error)
	GetTemplate(templateID templates.TemplateID) (template *templates.Template, err error)
	SaveTemplate(template *templates.Template) (err error)
	DeleteTemplate(templateID templates.TemplateID) (err error)

	PutClientUpdate(cl *clientupdates.Client) (err error)
	PutMeasurement(m *measures.Measure) (err error)

	LoadLatestRuleSet() (err error)
	SetRuleSet(rs *rules.RuleSet)
	GetLatestRuleActionStates(limit int) (states []*rules.RuleActionState, err error)
}
