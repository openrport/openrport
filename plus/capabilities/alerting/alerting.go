package alertingcap

import (
	"context"
	"errors"
	"time"

	"go.etcd.io/bbolt"

	"github.com/openrport/openrport/plus/capabilities/alerting/entities/clientupdates"
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/measures"
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/rules"
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/rundata"
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/templates"
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/validations"
	"github.com/openrport/openrport/server/notifications"
	"github.com/openrport/openrport/share/logger"
)

const NoLimit = -1

var (
	ErrEntityNotFound          = errors.New("entity not found")
	ErrInboundUpdatesQueueFull = errors.New("measures update queue full")
)

type CapabilityEx interface {
	Init(db *bbolt.DB) (err error)

	GetService() (as Service)

	RunRulesTest(ctx context.Context, runData *rundata.RunData, l *logger.Logger) (
		results *rundata.TestResults, errs validations.ErrorList, err error)
}

type Config struct {
	MaxWorkers    int    `mapstructure:"max_alerting_workers"`
	AlertsLogPath string `mapstructure:"alert_log_path"`
}

type Service interface {
	Run(ctx context.Context, scriptsDir string, notificationDispatcher notifications.Dispatcher, maxWorkers int)
	Stop() (err error)
	LoadDefaultRuleSet() (err error)

	PutClientUpdate(cl *clientupdates.Client) (err error)
	PutMeasurement(m *measures.Measure) (err error)

	GetAllTemplates() (templateList templates.TemplateList, err error)
	GetTemplate(templateID templates.TemplateID) (template *templates.Template, err error)
	SaveTemplate(template *templates.Template) (errs validations.ErrorList, err error)
	DeleteTemplate(templateID templates.TemplateID) (err error)

	LoadRuleSet(ruleSetID rules.RuleSetID) (rs *rules.RuleSet, err error)
	SaveRuleSet(rs *rules.RuleSet) (errs validations.ErrorList, err error)
	DeleteRuleSet(ruleSetID rules.RuleSetID) (err error)

	GetProblem(pid rules.ProblemID) (problem *rules.Problem, err error)
	GetLatestProblem(rid rules.RuleID, clientID string) (problem *rules.Problem, err error)
	SetProblemActive(pid rules.ProblemID) (err error)
	SetProblemResolved(pid rules.ProblemID, resolvedAt time.Time) (err error)
	GetLatestProblems(limit int) (problems []*rules.Problem, err error)

	GetSampleData(choice string) (sampleData *rundata.SampleData, err error)
}
