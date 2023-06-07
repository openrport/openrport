package chserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rportplus "github.com/realvnc-labs/rport/plus"
	licensecap "github.com/realvnc-labs/rport/plus/capabilities/license"
	"github.com/realvnc-labs/rport/plus/capabilities/license/licensemock"
	"github.com/realvnc-labs/rport/plus/capabilities/status"
	"github.com/realvnc-labs/rport/plus/capabilities/status/statusmock"
	"github.com/realvnc-labs/rport/server/api/authorization"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/security"
)

const (
	plusMockStatusCapability  = "plus-status-mock"
	plusMockLicenseCapability = "plus-license-mock"
)

type StatusInfoResponse struct {
	Data status.PlusStatusInfo
}

type plusManagerForMockStatus struct {
	cap map[string]rportplus.Capability

	rportplus.ManagerProvider
}

func (pm *plusManagerForMockStatus) RegisterCapability(capName string, newCap rportplus.Capability) (cap rportplus.Capability, err error) {
	if pm.cap == nil {
		pm.cap = make(map[string]rportplus.Capability)
	}
	newCap.InitProvider(nil)
	pm.cap[capName] = newCap
	return newCap, nil
}

func (pm *plusManagerForMockStatus) GetStatusCapabilityEx() (capEx status.CapabilityEx) {
	c := pm.cap[plusMockStatusCapability].(*statusmock.Capability)
	capEx = c.GetStatusCapabilityEx()
	return capEx
}

func (pm *plusManagerForMockStatus) GetLicenseCapabilityEx() (capEx licensecap.CapabilityEx) {
	c := pm.cap[plusMockLicenseCapability].(*licensemock.Capability)
	capEx = c.GetLicenseCapabilityEx()
	return capEx
}

func setupPlusStatus() (plusManager rportplus.Manager, plusConfig *rportplus.PlusConfig, plusLog *logger.Logger) {
	plusLog = logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	plusConfig = &rportplus.PlusConfig{
		PluginConfig: &rportplus.PluginConfig{
			PluginPath: defaultPluginPath,
		},
	}

	plusManager = &plusManagerForMockStatus{}
	plusManager.InitPlusManager(plusConfig, nil, plusLog)

	return plusManager, plusConfig, plusLog
}

func setupTestAPIListenerForStatus(
	t *testing.T,
	plusManager rportplus.Manager,
	plusConfig *rportplus.PlusConfig,
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
		tokenManager: mockTokenManager,
		bannedUsers:  security.NewBanList(0),
		userService:  mockUsersService,
		apiSessions:  newEmptyAPISessionCache(t),
	}
	al.initRouter()

	return al
}

func TestHandleGetPluginStatusInfoWhenValidLicense(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusStatus()

	_, err := plusManager.RegisterCapability(plusMockStatusCapability, &statusmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	licensemock.HasValidLicense = true

	_, err = plusManager.RegisterCapability(plusMockLicenseCapability, &licensemock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForStatus(t, plusManager, plusConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/plus/status", nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var statusInfo StatusInfoResponse
	err = json.NewDecoder(w.Body).Decode(&statusInfo)
	assert.NoError(t, err)

	assert.True(t, statusInfo.Data.IsEnabled)
	assert.True(t, statusInfo.Data.ValidLicense)
	assert.False(t, statusInfo.Data.IsTrial)
	assert.Equal(t, 2000, statusInfo.Data.LicenseInfo.MaxClients)
	assert.Equal(t, 50, statusInfo.Data.LicenseInfo.MaxUsers)
}

func TestHandleGetPluginStatusInfoWhenPlusEnabledButNoLicense(t *testing.T) {
	plusManager, plusConfig, plusLog := setupPlusStatus()

	_, err := plusManager.RegisterCapability(plusMockStatusCapability, &statusmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	licensemock.HasValidLicense = false

	_, err = plusManager.RegisterCapability(plusMockLicenseCapability, &licensemock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al := setupTestAPIListenerForStatus(t, plusManager, plusConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/plus/status", nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var statusInfo StatusInfoResponse
	err = json.NewDecoder(w.Body).Decode(&statusInfo)
	assert.NoError(t, err)

	assert.True(t, statusInfo.Data.IsEnabled)
	assert.False(t, statusInfo.Data.ValidLicense)
	assert.True(t, statusInfo.Data.IsTrial)
	assert.Equal(t, 0, statusInfo.Data.LicenseInfo.MaxClients)
	assert.Equal(t, 0, statusInfo.Data.LicenseInfo.MaxUsers)
}

func TestHandleGetPluginStatusInfoWhenPlusNotEnabled(t *testing.T) {
	var plusManager rportplus.Manager
	var plusConfig *rportplus.PlusConfig

	al := setupTestAPIListenerForStatus(t, plusManager, plusConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/plus/status", nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var statusInfo StatusInfoResponse
	err := json.NewDecoder(w.Body).Decode(&statusInfo)
	assert.NoError(t, err)

	assert.False(t, statusInfo.Data.IsEnabled)
	assert.False(t, statusInfo.Data.ValidLicense)
	assert.True(t, statusInfo.Data.IsTrial)
	assert.Equal(t, 0, statusInfo.Data.LicenseInfo.MaxClients)
	assert.Equal(t, 0, statusInfo.Data.LicenseInfo.MaxUsers)
}
