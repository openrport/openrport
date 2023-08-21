package chserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rportplus "github.com/realvnc-labs/rport/plus"
	alertingcap "github.com/realvnc-labs/rport/plus/capabilities/alerting"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/alertingmock"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rules"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rundata"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/templates"
	"github.com/realvnc-labs/rport/server/api/authorization"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/server/routes"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/security"
)

const (
	plusMockAlertingCapability = "plus-alerting-mock"
)

// TODO: (rs): these could probably be implemented with generics, but not a priority

type TemplateResponse struct {
	Data templates.Template
}

type TemplatesResponse struct {
	Data templates.TemplateList
}

type RuleSetResponse struct {
	Data rules.RuleSet
}

type ProblemResponse struct {
	Data rules.Problem
}

type ProblemsResponse struct {
	Data []*rules.Problem
}

type SampleDataResponse struct {
	Data *rundata.SampleData
}

type plusManagerForMockAlerting struct {
	cap map[string]rportplus.Capability

	rportplus.ManagerProvider
}

func (pm *plusManagerForMockAlerting) RegisterCapability(capName string, newCap rportplus.Capability) (cap rportplus.Capability, err error) {
	if pm.cap == nil {
		pm.cap = make(map[string]rportplus.Capability)
	}
	newCap.InitProvider(nil)
	pm.cap[capName] = newCap
	return newCap, nil
}

func (pm *plusManagerForMockAlerting) GetAlertingCapabilityEx() (capEx alertingcap.CapabilityEx) {
	c := pm.cap[plusMockAlertingCapability].(*alertingmock.Capability)
	capEx = c.GetAlertingCapabilityEx()
	return capEx
}

func setupPlusAlerting() (plusManager *plusManagerForMockAlerting, plusConfig *rportplus.PlusConfig, plusLog *logger.Logger) {
	plusLog = logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	plusConfig = &rportplus.PlusConfig{
		PluginConfig: &rportplus.PluginConfig{
			PluginPath: defaultPluginPath,
		},
	}

	plusManager = &plusManagerForMockAlerting{}
	plusManager.InitPlusManager(plusConfig, nil, plusLog)

	return plusManager, plusConfig, plusLog
}

func setupTestAPIListenerForAlerting(
	t *testing.T,
	plusManager *plusManagerForMockAlerting,
	plusConfig *rportplus.PlusConfig,
	plusLog *logger.Logger,
) (al *APIListener) {
	t.Helper()
	userGroup := "Administrators"
	user := &users.User{
		Username: "user1",
		Password: "$2y$05$ep2DdPDeLDDhwRrED9q/vuVEzRpZtB5WHCFT7YbcmH9r9oNmlsZOm",
	}
	mockUsersService := &MockUsersService{
		UserService: users.NewAPIService(users.NewStaticProvider([]*users.User{user}), false, 0, -1),
	}
	mockTokenManager := authorization.NewManager(
		CommonAPITokenTestDb(t, "user1", "prefixtkn", "the name",
			authorization.APITokenReadWrite,
			"mynicefi-xedl-enth-long-livedpasswor")) // APIToken database

	if plusConfig == nil {
		plusConfig = &rportplus.PlusConfig{}
	}

	al = &APIListener{
		insecureForTests: true,
		Server: &Server{
			config: &chconfig.Config{
				API: chconfig.APIConfig{
					DefaultUserGroup: userGroup,
				},
				PlusConfig: *plusConfig,
			},
			plusManager: plusManager,
		},
		Logger:       plusLog,
		tokenManager: mockTokenManager,
		bannedUsers:  security.NewBanList(0),
		userService:  mockUsersService,
		apiSessions:  newEmptyAPISessionCache(t),
	}
	al.initRouter()

	return al
}

func setup(t *testing.T) (al *APIListener, mockAS *alertingmock.MockServiceProvider) {
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al = setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

	mockAS = plusManager.GetAlertingCapabilityEx().GetService().(*alertingmock.MockServiceProvider)
	require.NotNil(t, mockAS)

	return al, mockAS
}

func TestShouldErrorWhenRuleSetNotFound(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute+"/missing", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, res.StatusCode)
	}
}

func TestShouldReturnDefaultRuleSet(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute, nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	var ruleSetInfo RuleSetResponse
	err := json.NewDecoder(w.Body).Decode(&ruleSetInfo)
	assert.NoError(t, err)
}

func TestShouldSaveRuleSet(t *testing.T) {
	al, mockAS := setup(t)

	// remove the initial latest ruleset
	err := mockAS.DeleteRuleSet(rules.DefaultRuleSetID)
	require.NoError(t, err)

	defaultRS := rules.RuleSet{
		Rules: []rules.Rule{
			{
				ID: "rule-id",
			},
		},
	}

	defaultRSJSON, err := json.Marshal(defaultRS)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute, bytes.NewReader(defaultRSJSON))

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	savedRS, ok := mockAS.RuleSets[rules.DefaultRuleSetID]
	require.True(t, ok)

	// very basic check
	assert.Equal(t, defaultRS.Rules[0].ID, savedRS.Rules[0].ID)
}

func TestShouldDeleteRuleSet(t *testing.T) {
	al, mockAS := setup(t)

	// default rule set already part of the test data

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute, nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	// the default rule set in the mock rule sets should not be present
	_, ok := mockAS.RuleSets[rules.DefaultRuleSetID]
	require.False(t, ok)
}

func TestShouldOnGetErrorWhenTemplateNotFound(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute+"/missing", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, res.StatusCode)
	}
}

func TestShouldReturnTemplate(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute+"/t1", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	var templateInfo TemplateResponse
	err := json.NewDecoder(w.Body).Decode(&templateInfo)
	assert.NoError(t, err)

	assert.Equal(t, templates.TemplateID("t1"), templateInfo.Data.ID)
}

func TestShouldSaveTemplate(t *testing.T) {
	al, mockAS := setup(t)

	t1, err := mockAS.GetTemplate("t1")
	require.NoError(t, err)

	t10 := *t1
	t10.ID = ""

	t10JSON, err := json.Marshal(t10)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute+"/t10", bytes.NewReader(t10JSON))

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	savedTemplate, ok := mockAS.Templates["t10"]
	require.True(t, ok)

	assert.Equal(t, templates.TemplateID("t10"), savedTemplate.ID)
}

func TestShouldDeleteTemplate(t *testing.T) {
	al, mockAS := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute+"/t1", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	// the test rule set in the mock saved templates should not be present
	_, ok := mockAS.Templates["t1"]
	require.False(t, ok)
}

func TestShouldNotDeleteActiveTemplate(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute+"/t2", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestShouldGetAllTemplates(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute, nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	var templatesInfo TemplatesResponse
	err := json.NewDecoder(w.Body).Decode(&templatesInfo)
	assert.NoError(t, err)

	assert.Equal(t, templates.TemplateID("t1"), templatesInfo.Data[0].ID)
	assert.Equal(t, templates.TemplateID("t2"), templatesInfo.Data[1].ID)
}

func TestShouldOnGetErrorWhenProblemNotFound(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"/missing", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, res.StatusCode)
	}
}

func TestShouldGetProblem(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"/p1", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	var problemInfo ProblemResponse
	err := json.NewDecoder(w.Body).Decode(&problemInfo)
	assert.NoError(t, err)

	assert.Equal(t, rules.ProblemID("p1"), problemInfo.Data.ID)
}

func TestShouldSaveProblemResolved(t *testing.T) {
	al, mockAS := setup(t)

	updateRequest := rules.ProblemUpdateRequest{
		Active: false,
	}

	updateRequestJSON, err := json.Marshal(updateRequest)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"/p1", bytes.NewReader(updateRequestJSON))

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	savedProblem, ok := mockAS.Problems["p1"]
	require.True(t, ok)

	assert.Equal(t, rules.ProblemID("p1"), savedProblem.ID)
	assert.Equal(t, false, savedProblem.Active)
}

func getProblemsInfo(t *testing.T, w *httptest.ResponseRecorder) (problemsInfo ProblemsResponse) {
	err := json.NewDecoder(w.Body).Decode(&problemsInfo)
	assert.NoError(t, err)
	return problemsInfo
}

func TestShouldGetLatestProblems(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute, nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	problemsInfo := getProblemsInfo(t, w)

	assert.Equal(t, 3, len(problemsInfo.Data))
	assert.Equal(t, rules.ProblemID("p3"), problemsInfo.Data[0].ID)
	assert.Equal(t, rules.ProblemID("p2"), problemsInfo.Data[1].ID)
	assert.Equal(t, rules.ProblemID("p1"), problemsInfo.Data[2].ID)
}

func TestShouldGetLatestProblemsWithProblemIDFilter(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"?filter[problem_id]=p2", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	problemsInfo := getProblemsInfo(t, w)

	assert.Equal(t, 1, len(problemsInfo.Data))
	assert.Equal(t, rules.ProblemID("p2"), problemsInfo.Data[0].ID)
}

func TestShouldGetLatestProblemsWithProblemActiveFilter(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"?filter[active]=true", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	problemsInfo := getProblemsInfo(t, w)

	assert.Equal(t, 1, len(problemsInfo.Data))
	assert.Equal(t, rules.ProblemID("p1"), problemsInfo.Data[0].ID)
}

func TestShouldGetLatestProblemsWithProblemsNotActiveFilter(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"?filter[active]=false", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	problemsInfo := getProblemsInfo(t, w)

	assert.Equal(t, 2, len(problemsInfo.Data))
	assert.Equal(t, rules.ProblemID("p3"), problemsInfo.Data[0].ID)
	assert.Equal(t, rules.ProblemID("p2"), problemsInfo.Data[1].ID)
}

func TestShouldGetLatestProblemsGreaterThanDateFilter(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"?filter[created_at][gt]="+"2023-06-30T00:00:00Z", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	problemsInfo := getProblemsInfo(t, w)

	assert.Equal(t, 2, len(problemsInfo.Data))
	assert.Equal(t, rules.ProblemID("p3"), problemsInfo.Data[0].ID)
	assert.Equal(t, rules.ProblemID("p2"), problemsInfo.Data[1].ID)
}

func TestShouldGetLatestProblemsLessThanDateFilter(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"?filter[created_at][lt]="+"2023-06-30T00:00:00Z", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	problemsInfo := getProblemsInfo(t, w)

	assert.Equal(t, 1, len(problemsInfo.Data))
	assert.Equal(t, rules.ProblemID("p1"), problemsInfo.Data[0].ID)
}

func TestShouldGetLatestProblemsEqualDateFilter(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"?filter[created_at][eq]=2023-06-03", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	problemsInfo := getProblemsInfo(t, w)

	assert.Equal(t, 1, len(problemsInfo.Data))
	assert.Equal(t, rules.ProblemID("p1"), problemsInfo.Data[0].ID)
}

func TestShouldGetLatestProblemsWithSort(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASProblemsRoute+"?sort=-rule_id", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	problemsInfo := getProblemsInfo(t, w)

	assert.Equal(t, 3, len(problemsInfo.Data))
	assert.Equal(t, rules.RuleID("r2"), problemsInfo.Data[0].RuleID)
	assert.Equal(t, rules.RuleID("r1"), problemsInfo.Data[1].RuleID)
	assert.Equal(t, rules.RuleID("r1"), problemsInfo.Data[2].RuleID)
}

func getSampleDataInfo(t *testing.T, w *httptest.ResponseRecorder) (sampleDataInfo SampleDataResponse) {
	err := json.NewDecoder(w.Body).Decode(&sampleDataInfo)
	assert.NoError(t, err)
	return sampleDataInfo
}

func TestShouldGetSampleDataWindows(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET",
		routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute+routes.ASSampleDataRoute+"/windows",
		nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		require.Equal(t, http.StatusOK, res.StatusCode)
	}

	sampleDataInfo := getSampleDataInfo(t, w)
	assert.NotZero(t, sampleDataInfo.Data)
	sampleData := sampleDataInfo.Data
	assert.Equal(t, "windows", sampleData.CL[0].ID)
	assert.Equal(t, "windows", sampleData.M[0].ClientID)
}

func TestShouldGetSampleDataLinux(t *testing.T) {
	al, _ := setup(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET",
		routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute+routes.ASSampleDataRoute+"/linux",
		nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		require.Equal(t, http.StatusOK, res.StatusCode)
	}

	sampleDataInfo := getSampleDataInfo(t, w)
	assert.NotZero(t, sampleDataInfo.Data)
	sampleData := sampleDataInfo.Data
	assert.Equal(t, "linux", sampleData.CL[0].ID)
	assert.Equal(t, "linux", sampleData.M[0].ClientID)
}
