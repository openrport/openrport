package alertingmock

import (
	"context"
	"plugin"
	"sort"
	"time"

	"github.com/dgraph-io/badger/v4"
	"go.etcd.io/bbolt"

	alertingcap "github.com/realvnc-labs/rport/plus/capabilities/alerting"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/clientupdates"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/measures"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rules"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/templates"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/validations"
	"github.com/realvnc-labs/rport/plus/capabilities/status"
	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/types"
)

type MockCapabilityProvider struct {
	serviceMock *MockServiceProvider
}

type Capability struct {
	Provider *MockCapabilityProvider

	Config *status.Config
	Logger *logger.Logger
}

// GetInitFuncName return the empty string as the mock capability doesn't use the plugin
func (cap *Capability) GetInitFuncName() (name string) {
	return ""
}

// InitProvider sets the capability provider to the local mock implementation
func (cap *Capability) InitProvider(initFn plugin.Symbol) {
	if cap.Provider == nil {
		cap.Provider = &MockCapabilityProvider{
			serviceMock: NewMockServiceProvider(),
		}
	}
}

// GetAlertingCapabilityEx returns the mock provider's interface to the capability
// functions
func (cap *Capability) GetAlertingCapabilityEx() (capEx alertingcap.CapabilityEx) {
	return cap.Provider
}

// GetConfigValidator returns a validator interface that can be called to
// validate the capability config
func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return cap.Provider
}

// ValidateConfig does nothing for the mock implementation
func (mp *MockCapabilityProvider) ValidateConfig() (err error) {
	return nil
}

// GetService returns a mock service
func (mp *MockCapabilityProvider) InitBadgerDB(_ *badger.DB) (err error) {
	return nil
}

func (mp *MockCapabilityProvider) Init(_ *bbolt.DB) (err error) {
	return nil
}

// GetService returns a mock service
func (mp *MockCapabilityProvider) GetService() (s alertingcap.Service) {
	if mp.serviceMock == nil {
		mp.serviceMock = &MockServiceProvider{}
	}
	return mp.serviceMock
}

func newTestTemplates() map[templates.TemplateID]templates.Template {
	testTemplates := map[templates.TemplateID]templates.Template{
		"t1": {
			ID:         "t1",
			Transport:  "smtp",
			Subject:    "{{.Outcome}} for {{.Rule.ID}} SUBJECT1",
			Body:       "The client with ID: {{.Client.ID}} has triggered rule ID: {{.Rule.ID}} BODY1",
			HTML:       false,
			Recipients: []string{"t1@test.com", "t2@test.com"},
		},
		"t2": {
			ID:         "t2",
			Transport:  "script",
			Subject:    "{{.Outcome}} for {{.Rule.ID}} SUBJECT2",
			Body:       "The client with ID: {{.Client.ID}} has triggered rule ID: {{.Rule.ID}} BODY2",
			HTML:       true,
			Recipients: []string{"t3@test.com", "t4@test.com"},
		},
		"t3": {
			ID:         "t3",
			Transport:  "smtp",
			Subject:    "{{.Outcome}} for {{.Rule.ID}} SUBJECT3",
			Body:       "The client with ID: {{.Client.ID}} has triggered rule ID: {{.Rule.ID}} BODY3",
			HTML:       false,
			Recipients: []string{"t5@test.com", "t6@test.com"},
		},
	}
	return testTemplates
}

func newTestRuleSets() map[rules.RuleSetID]rules.RuleSet {
	al := rules.ActionList{
		rules.Action{
			NotifyList: &rules.NotifyList{
				"t1",
				"t2",
				"t3",
			},
		},
	}

	latest := rules.RuleSet{
		RuleSetID: rules.DefaultRuleSetID,
		Rules: []rules.Rule{
			{
				ID:      "rule1",
				Ex:      "true",
				Actions: al,
			},
		},
	}

	testRuleSets := map[rules.RuleSetID]rules.RuleSet{
		rules.DefaultRuleSetID: latest,
	}

	return testRuleSets
}

func newTestProblems() map[rules.ProblemID]rules.Problem {
	testProblems := map[rules.ProblemID]rules.Problem{
		"p1": {
			ID:     "p1",
			RuleID: "r1",
		},
		"p2": {
			ID:     "p2",
			RuleID: "r1",
		},
		"p3": {
			ID:     "p3",
			RuleID: "r2",
		},
	}
	return testProblems
}

type MockServiceProvider struct {
	RuleSets  map[rules.RuleSetID]rules.RuleSet
	Templates map[templates.TemplateID]templates.Template
	Problems  map[rules.ProblemID]rules.Problem
}

func NewMockServiceProvider() (mp *MockServiceProvider) {
	mp = &MockServiceProvider{
		Templates: newTestTemplates(),
		RuleSets:  newTestRuleSets(),
		Problems:  newTestProblems(),
	}
	return mp
}

func (mp *MockServiceProvider) Run(ctx context.Context, _ notifications.Dispatcher) {
}

func (mp *MockServiceProvider) Stop() (err error) {
	return nil
}

func (mp *MockServiceProvider) LoadRuleSet(ruleSetID rules.RuleSetID) (rs *rules.RuleSet, err error) {
	testRS, ok := mp.RuleSets[ruleSetID]
	if !ok {
		return nil, alertingcap.ErrEntityNotFound
	}
	return &testRS, nil
}

func (mp *MockServiceProvider) SaveRuleSet(rs *rules.RuleSet) (errs validations.ErrorList, err error) {
	mp.RuleSets[rs.RuleSetID] = *rs
	return nil, nil
}

func (mp *MockServiceProvider) DeleteRuleSet(ruleSetID rules.RuleSetID) (err error) {
	delete(mp.RuleSets, ruleSetID)
	return nil
}

func (mp *MockServiceProvider) GetTemplate(templateID templates.TemplateID) (template *templates.Template, err error) {
	testTemplates := newTestTemplates()
	tt, ok := testTemplates[templateID]
	if !ok {
		return nil, alertingcap.ErrEntityNotFound
	}
	return &tt, nil
}

func (mp *MockServiceProvider) GetAllTemplates() (templateList templates.TemplateList, err error) {
	for _, template := range mp.Templates {
		t := template
		templateList = append(templateList, &t)
	}
	sort.Slice(templateList, func(a int, b int) bool {
		return templateList[a].ID < templateList[b].ID
	})
	return templateList, nil
}

func (mp *MockServiceProvider) SaveTemplate(template *templates.Template) (errs validations.ErrorList, err error) {
	mp.Templates[template.ID] = *template
	return nil, nil
}

func (mp *MockServiceProvider) DeleteTemplate(templateID templates.TemplateID) (err error) {
	// simulate a template failing to delete due to being active
	if templateID == "t2" {
		return templates.ErrTemplateInUse
	}
	delete(mp.Templates, templateID)
	return nil
}

func (mp *MockServiceProvider) PutClientUpdate(cl *clientupdates.Client) (err error) {
	return nil
}

func (mp *MockServiceProvider) PutMeasurement(m *measures.Measure) (err error) {
	return nil
}

func (mp *MockServiceProvider) LoadDefaultRuleSet() (err error) {
	return nil
}

func (mp *MockServiceProvider) SetRuleSet(rs *rules.RuleSet) {
}

func (mp *MockServiceProvider) GetProblem(pid rules.ProblemID) (problem *rules.Problem, err error) {
	if pid == "p1" {
		return &rules.Problem{
			ID: "p1",
		}, nil
	}
	return problem, nil
}

func (mp *MockServiceProvider) GetLatestProblem(rid rules.RuleID, clientID string) (problem *rules.Problem, err error) {
	return problem, nil
}

func (mp *MockServiceProvider) SetProblemActive(pid rules.ProblemID) (err error) {
	return nil
}

func (mp *MockServiceProvider) SetProblemResolved(pid rules.ProblemID, resolvedAt time.Time) (err error) {
	problem, ok := mp.Problems[pid]
	if !ok {
		return alertingcap.ErrEntityNotFound
	}
	problem.Active = false
	problem.ResolvedAt = types.NewTimeJSON(resolvedAt)
	mp.Problems[pid] = problem
	return nil
}

func (mp *MockServiceProvider) GetLatestProblems(limit int) (problems []*rules.Problem, err error) {
	for _, problem := range mp.Problems {
		p := problem
		problems = append(problems, &p)
	}
	sort.Slice(problems, func(a, b int) bool {
		return problems[a].ID < problems[b].ID
	})
	return problems, nil
}
