package alertingmock

import (
	"context"
	"plugin"
	"sort"

	"github.com/dgraph-io/badger/v4"

	alertingcap "github.com/realvnc-labs/rport/plus/capabilities/alerting"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/actions"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/clientupdates"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/measures"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rules"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/templates"
	"github.com/realvnc-labs/rport/plus/capabilities/status"
	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/logger"
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
		cap.Provider = &MockCapabilityProvider{}
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
func (mp *MockCapabilityProvider) Init(_ *badger.DB) (err error) {
	return nil
}

// GetService returns a mock service
func (mp *MockCapabilityProvider) GetService() (s alertingcap.Service) {
	if mp.serviceMock == nil {
		mp.serviceMock = &MockServiceProvider{}
	}
	return mp.serviceMock
}

type MockServiceProvider struct {
	SavedRuleSets  map[rules.RuleSetID]rules.RuleSet
	SavedTemplates map[templates.TemplateID]templates.Template
}

func (mp *MockServiceProvider) Run(ctx context.Context) {
}

func (mp *MockServiceProvider) Stop() (err error) {
	return nil
}

func (mp *MockServiceProvider) LoadRuleSet(ruleSetID rules.RuleSetID) (rs *rules.RuleSet, err error) {
	al := rules.ActionList{
		rules.Action{
			ActType: actions.NotifyActionType,
			NotifyAction: &rules.NotifyAction{
				TemplateIDs: []templates.TemplateID{"t1", "t2", "t3"},
			},
		},
	}

	rs1 := rules.RuleSet{
		RuleSetID: "rs1",
		Rules: []rules.Rule{
			{
				ID:      "rule1",
				Ex:      "true",
				Actions: al,
			},
		},
	}

	testRuleSets := map[rules.RuleSetID]rules.RuleSet{
		"rs1": rs1,
	}

	testRS, ok := testRuleSets[ruleSetID]
	if !ok {
		return nil, rules.ErrRuleSetNotFound
	}
	return &testRS, nil
}

func (mp *MockServiceProvider) SaveRuleSet(rs *rules.RuleSet) (err error) {
	if mp.SavedRuleSets == nil {
		mp.SavedRuleSets = make(map[rules.RuleSetID]rules.RuleSet, 16)
	}
	mp.SavedRuleSets[rs.RuleSetID] = *rs
	return nil
}

func (mp *MockServiceProvider) DeleteRuleSet(ruleSetID rules.RuleSetID) (err error) {
	delete(mp.SavedRuleSets, ruleSetID)
	return nil
}

func (mp *MockServiceProvider) GetTemplate(templateID templates.TemplateID) (template *templates.Template, err error) {
	var testTemplates = map[templates.TemplateID]templates.Template{
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
	tt, ok := testTemplates[templateID]
	if !ok {
		return nil, templates.ErrTemplateNotFound
	}
	return &tt, nil
}

func (mp *MockServiceProvider) GetAllTemplates() (templateList templates.TemplateList, err error) {
	for _, template := range mp.SavedTemplates {
		t := template
		templateList = append(templateList, &t)
	}
	sort.Slice(templateList, func(a int, b int) bool {
		return templateList[a].ID < templateList[b].ID
	})
	return templateList, nil
}

func (mp *MockServiceProvider) SaveTemplate(template *templates.Template) (err error) {
	if mp.SavedTemplates == nil {
		mp.SavedTemplates = make(map[templates.TemplateID]templates.Template, 16)
	}
	mp.SavedTemplates[template.ID] = *template
	return nil
}

func (mp *MockServiceProvider) DeleteTemplate(templateID templates.TemplateID) (err error) {
	delete(mp.SavedTemplates, templateID)
	return nil
}

func (mp *MockServiceProvider) PutClientUpdate(cl *clientupdates.Client) (err error) {
	return nil
}

func (mp *MockServiceProvider) PutMeasurement(m *measures.Measure) (err error) {
	return nil
}

func (mp *MockServiceProvider) LoadLatestRuleSet() (err error) {
	return nil
}

func (mp *MockServiceProvider) SetRuleSet(rs *rules.RuleSet) {
}

func (mp *MockServiceProvider) GetLatestRuleActionStates(limit int) (states []*rules.RuleActionState, err error) {
	return nil, nil
}
