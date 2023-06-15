package chserver

import (
	"bytes"
	"encoding/json"
	"fmt"
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

type TemplateResponse struct {
	Data templates.Template
}

type TemplatesResponse struct {
	Data templates.TemplateList
}

type RuleSetResponse struct {
	Data rules.RuleSet
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

func TestShouldErrorWhenRuleSetNotFound(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute+"/missing", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, res.StatusCode)
	}
}

func TestShouldReturnRuleSet(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute+"/rs1", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	var ruleSetInfo RuleSetResponse
	err = json.NewDecoder(w.Body).Decode(&ruleSetInfo)
	assert.NoError(t, err)

	assert.Equal(t, rules.RuleSetID("rs1"), ruleSetInfo.Data.RuleSetID)
}

func TestShouldSaveRuleSet(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

	mockAS := plusManager.GetAlertingCapabilityEx().GetService().(*alertingmock.MockServiceProvider)
	require.NotNil(t, mockAS)

	rs1, err := mockAS.LoadRuleSet("rs1")
	require.NoError(t, err)

	rs2 := *rs1
	rs2.RuleSetID = "rs2"

	rs2JSON, err := json.Marshal(rs2)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute, bytes.NewReader(rs2JSON))

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	savedRS, ok := mockAS.SavedRuleSets["rs2"]
	require.True(t, ok)

	assert.Equal(t, rs2.RuleSetID, savedRS.RuleSetID)
}

func TestShouldDeleteRuleSet(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

	mockAS := plusManager.GetAlertingCapabilityEx().GetService().(*alertingmock.MockServiceProvider)
	require.NotNil(t, mockAS)

	// load a rule set from the mock for the test
	rs1, err := mockAS.LoadRuleSet("rs1")
	require.NoError(t, err)

	// save the test rule set in the mock saved rule sets
	err = mockAS.SaveRuleSet(rs1)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASRuleSetRoute+"/rs1", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	// the test rule set in the mock saved rule sets should not be present
	_, ok := mockAS.SavedRuleSets["rs2"]
	require.False(t, ok)
}

func TestShouldErrorWhenTemplateNotFound(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

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
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute+"/t1", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	fmt.Printf("%v\n", w.Body.String())

	var templateInfo TemplateResponse
	err = json.NewDecoder(w.Body).Decode(&templateInfo)
	assert.NoError(t, err)

	assert.Equal(t, templates.TemplateID("t1"), templateInfo.Data.ID)
}

func TestShouldSaveTemplate(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

	mockAS := plusManager.GetAlertingCapabilityEx().GetService().(*alertingmock.MockServiceProvider)
	require.NotNil(t, mockAS)

	t1, err := mockAS.GetTemplate("t1")
	require.NoError(t, err)

	t10 := *t1
	t10.ID = "t10"

	t10JSON, err := json.Marshal(t10)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute, bytes.NewReader(t10JSON))

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	mockAS = plusManager.GetAlertingCapabilityEx().GetService().(*alertingmock.MockServiceProvider)
	require.NotNil(t, mockAS)

	savedTemplate, ok := mockAS.SavedTemplates["t10"]
	require.True(t, ok)

	assert.Equal(t, t10.ID, savedTemplate.ID)
}

func TestShouldDeleteTemplate(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

	mockAS := plusManager.GetAlertingCapabilityEx().GetService().(*alertingmock.MockServiceProvider)
	require.NotNil(t, mockAS)

	// load a template from the mock for the test
	t1, err := mockAS.GetTemplate("t1")
	require.NoError(t, err)

	// save the test template in the mock saved templates
	err = mockAS.SaveTemplate(t1)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute+"/t1", nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	// the test rule set in the mock saved templates should not be present
	_, ok := mockAS.SavedTemplates["t1"]
	require.False(t, ok)
}

func TestShouldGetAllTemplates(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusAlerting()

	_, err := plusManager.RegisterCapability(plusMockAlertingCapability, &alertingmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForAlerting(t,
		plusManager,
		plusConfig,
		plusLog)

	mockAS := plusManager.GetAlertingCapabilityEx().GetService().(*alertingmock.MockServiceProvider)
	require.NotNil(t, mockAS)

	// load a template from the mock for the test
	t1, err := mockAS.GetTemplate("t1")
	require.NoError(t, err)

	// save the test template in the mock saved templates
	err = mockAS.SaveTemplate(t1)
	require.NoError(t, err)

	// load a template from the mock for the test
	t2, err := mockAS.GetTemplate("t2")
	require.NoError(t, err)

	// save the test template in the mock saved templates
	err = mockAS.SaveTemplate(t2)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AlertingServiceRoutesPrefix+routes.ASTemplatesRoute, nil)

	al.router.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	var templatesInfo TemplatesResponse
	err = json.NewDecoder(w.Body).Decode(&templatesInfo)
	assert.NoError(t, err)

	assert.Equal(t, templates.TemplateID("t1"), templatesInfo.Data[0].ID)
	assert.Equal(t, templates.TemplateID("t2"), templatesInfo.Data[1].ID)
}
