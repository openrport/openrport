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
	"github.com/realvnc-labs/rport/plus/capabilities/oauth"
	"github.com/realvnc-labs/rport/plus/capabilities/oauth/oauthmock"
	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/authorization"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/server/routes"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/security"
)

const (
	plusMockOAuthCapability = "plus-oauth-mock"
)

type plusManagerForMockOAuth struct {
	cap rportplus.Capability

	rportplus.ManagerProvider
}

func initMockPlusManager() (plus *plusManagerForMockOAuth) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	config := &chconfig.Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: rportplus.PlusConfig{
			PluginConfig: &rportplus.PluginConfig{
				PluginPath: defaultPluginPath,
			},
			OAuthConfig: &oauth.Config{
				Provider: oauth.GitHubOAuthProvider,
			},
		},
	}

	plus = &plusManagerForMockOAuth{}
	plus.InitPlusManager(&config.PlusConfig, nil, plusLog)
	return plus
}

func (pm *plusManagerForMockOAuth) RegisterCapability(capName string, newCap rportplus.Capability) (cap rportplus.Capability, err error) {
	newCap.InitProvider(nil)
	pm.cap = newCap
	return newCap, nil
}

func (pm *plusManagerForMockOAuth) GetOAuthCapabilityEx() (capEx oauth.CapabilityEx) {
	cap := pm.cap.(*oauthmock.Capability)
	capEx = cap.GetOAuthCapabilityEx()
	return capEx
}

func TestHandleFailedLoginWhenUsingOAuth(t *testing.T) {
	al, _ := setupAPIListenerForPlusOAuth(t,
		&oauth.Config{
			Provider:          oauth.GitHubOAuthProvider,
			PermittedUserList: true,
			ClientSecret:      "1234",
		},
		"user1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/login", nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var failedResponse api.ErrorPayload

	err := json.NewDecoder(w.Body).Decode(&failedResponse)
	assert.NoError(t, err)

	loginErrors := failedResponse.Errors
	loginError := loginErrors[0].Title
	assert.Contains(t, loginError, "authorization disabled")
}

func TestHandleGetOAuthProvider(t *testing.T) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	plusConfig := &rportplus.PlusConfig{
		PluginConfig: &rportplus.PluginConfig{
			PluginPath: defaultPluginPath,
		},
	}

	oauthConfig := &oauth.Config{
		Provider: oauth.GitHubOAuthProvider,
	}

	plusManager := initMockPlusManager()

	_, err := plusManager.RegisterCapability(plusMockOAuthCapability, &oauthmock.Capability{
		Logger: plusLog,
	})
	require.NoError(t, err)

	al, _ := setupTestAPIListenerForOAuth(t, plusManager, plusConfig, oauthConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+routes.AuthRoutesPrefix+routes.AuthProviderRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	info, err := GetSuccessPayloadResponse[AuthProviderInfo](w.Body)
	assert.NoError(t, err)

	assert.Equal(t, "github", info.AuthProvider)
	assert.Equal(t, routes.AllRoutesPrefix+routes.AuthRoutesPrefix+routes.AuthSettingsRoute, info.SettingsURI)
	assert.Equal(t, routes.AllRoutesPrefix+routes.AuthRoutesPrefix+routes.AuthDeviceSettingsRoute, info.DeviceSettingsURI)
}

type AuthSettingsResponse struct {
	Data AuthSettings
}

func setupPlusOAuth() (plusManager rportplus.Manager, plusConfig *rportplus.PlusConfig, oauthConfig *oauth.Config, plusLog *logger.Logger) {
	plusLog = logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	plusConfig = &rportplus.PlusConfig{
		PluginConfig: &rportplus.PluginConfig{
			PluginPath: defaultPluginPath,
		},
	}

	oauthConfig = &oauth.Config{
		Provider: oauth.GitHubOAuthProvider,
	}

	plusManager = &plusManagerForMockOAuth{}
	plusManager.InitPlusManager(plusConfig, nil, plusLog)

	return plusManager, plusConfig, oauthConfig, plusLog
}

func TestHandleGetAuthSettings(t *testing.T) {
	plusManager, plusConfig, oauthConfig, plusLog := setupPlusOAuth()

	_, err := plusManager.RegisterCapability(plusMockOAuthCapability, &oauthmock.Capability{
		Config: oauthConfig,
		Logger: plusLog,
	})
	require.NoError(t, err)

	al, _ := setupTestAPIListenerForOAuth(t, plusManager, plusConfig, oauthConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+routes.AuthRoutesPrefix+routes.AuthSettingsRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var settings AuthSettingsResponse
	err = json.NewDecoder(w.Body).Decode(&settings)
	assert.NoError(t, err)

	assert.Equal(t, "github", settings.Data.AuthProvider)
	assert.Equal(t, "mock login msg", settings.Data.LoginInfo.LoginMsg)
	assert.Equal(t, "mock authorize url", settings.Data.LoginInfo.AuthorizeURL)
	assert.Equal(t, "/mock_login_uri", settings.Data.LoginInfo.LoginURI)
}

type DeviceAuthSettingsResponse struct {
	Data DeviceAuthSettings
}

func TestHandleGetAuthDeviceSettings(t *testing.T) {
	plusManager, plusConfig, oauthConfig, plusLog := setupPlusOAuth()

	_, err := plusManager.RegisterCapability(plusMockOAuthCapability, &oauthmock.Capability{
		Config: oauthConfig,
		Logger: plusLog,
	})
	require.NoError(t, err)

	al, _ := setupTestAPIListenerForOAuth(t, plusManager, plusConfig, oauthConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+routes.AuthRoutesPrefix+routes.AuthDeviceSettingsRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var settings DeviceAuthSettingsResponse
	err = json.NewDecoder(w.Body).Decode(&settings)
	assert.NoError(t, err)

	assert.Equal(t, "github", settings.Data.AuthProvider)
	assert.Equal(t, "/mock_device_login_uri", settings.Data.LoginInfo.LoginURI)
	assert.Equal(t, "mock-user-code", settings.Data.LoginInfo.DeviceAuthInfo.UserCode)
	assert.Equal(t, "mock-device-code", settings.Data.LoginInfo.DeviceAuthInfo.DeviceCode)
	assert.Equal(t, "mock-verification-uri", settings.Data.LoginInfo.DeviceAuthInfo.VerificationURI)
	assert.Equal(t, 333, settings.Data.LoginInfo.DeviceAuthInfo.ExpiresIn)
	assert.Equal(t, 4, settings.Data.LoginInfo.DeviceAuthInfo.Interval)
	assert.Equal(t, "mock-message", settings.Data.LoginInfo.DeviceAuthInfo.Message)
}

func TestHandleGetAuthDeviceSettingsWithError(t *testing.T) {
	plusManager, plusConfig, oauthConfig, plusLog := setupPlusOAuth()

	mockOAuthCapability := &oauthmock.Capability{
		Config: oauthConfig,
		Logger: plusLog,
	}
	_, err := plusManager.RegisterCapability(plusMockOAuthCapability, mockOAuthCapability)
	require.NoError(t, err)

	al, _ := setupTestAPIListenerForOAuth(t, plusManager, plusConfig, oauthConfig)

	mockOAuthCapability.Provider.ShouldFailGetLoginInfo = true

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+routes.AuthRoutesPrefix+routes.AuthDeviceSettingsRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleGetAuthSettingsWhenFailedToGetLoginURL(t *testing.T) {
	plusManager, plusConfig, oauthConfig, plusLog := setupPlusOAuth()

	mockOAuthCapability := &oauthmock.Capability{
		Config: oauthConfig,
		Logger: plusLog,
	}

	_, err := plusManager.RegisterCapability(plusMockOAuthCapability, mockOAuthCapability)
	require.NoError(t, err)

	al, _ := setupTestAPIListenerForOAuth(t, plusManager, plusConfig, oauthConfig)

	mockOAuthCapability.Provider.ShouldFailGetLoginInfo = true

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+routes.AuthRoutesPrefix+routes.AuthSettingsRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleOAuthAuthorizationCode(t *testing.T) {
	tc := []struct {
		Name           string
		OAuthConfig    *oauth.Config
		Username       string
		ExpectedStatus int
	}{
		{
			Name: "unknown user",
			OAuthConfig: &oauth.Config{
				Provider:             oauth.GitHubOAuthProvider,
				RequiredOrganization: "cloudradar",
				PermittedUserList:    true, // don't create missing user
			},
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name: "create missing user",
			OAuthConfig: &oauth.Config{
				Provider:             oauth.GitHubOAuthProvider,
				RequiredOrganization: "cloudradar",
				PermittedUserList:    false, // create missing user
			},
			Username:       "added-user",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name: "use api auth with known user",
			OAuthConfig: &oauth.Config{
				Provider:          oauth.GitHubOAuthProvider,
				PermittedUserList: true,
			},
			Username:       "user1",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name: "use api auth with unknown user",
			OAuthConfig: &oauth.Config{
				Provider:          oauth.GitHubOAuthProvider,
				PermittedUserList: true,
			},
			Username:       "unknown-user",
			ExpectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tc {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			al, mockUsersService := setupAPIListenerForPlusOAuth(t, tc.OAuthConfig, tc.Username)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", routes.AllRoutesPrefix+oauth.DefaultLoginURI, nil)

			al.router.ServeHTTP(w, req)

			assert.Equal(t, tc.ExpectedStatus, w.Code)

			if w.Code == http.StatusOK {
				if tc.ExpectedStatus == http.StatusOK {
					assert.Contains(t, w.Body.String(), `{"data":{"token":"`)
				}
			}

			if !tc.OAuthConfig.PermittedUserList {
				changedUser := mockUsersService.ChangeUser
				assert.Equal(t, tc.Username, changedUser.Username)
			}
		})
	}
}

func TestShouldHandleGetDeviceAuth(t *testing.T) {
	plusManager, plusConfig, oauthConfig, plusLog := setupPlusOAuth()

	_, err := plusManager.RegisterCapability(plusMockOAuthCapability, &oauthmock.Capability{
		Config: oauthConfig,
		Logger: plusLog,
	})
	require.NoError(t, err)

	al, _ := setupTestAPIListenerForOAuth(t, plusManager, plusConfig, oauthConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+oauth.DefaultDeviceLoginURI, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Contains(t, w.Body.String(), `{"data":{"token":"`)
}

func TestShouldHandleGetDeviceAuthStatusWithError(t *testing.T) {
	plusManager, plusConfig, oauthConfig, plusLog := setupPlusOAuth()

	mockOAuthCapability := &oauthmock.Capability{
		Config: oauthConfig,
		Logger: plusLog,
	}

	_, err := plusManager.RegisterCapability(plusMockOAuthCapability, mockOAuthCapability)
	require.NoError(t, err)

	mockOAuthCapability.Provider.ShouldFailGetAccessTokenForDevice = true

	al, _ := setupTestAPIListenerForOAuth(t, plusManager, plusConfig, oauthConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+oauth.DefaultDeviceLoginURI, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	// note: As the GET to rport was successful and being used in a polling context
	// we're returning a success response, but the provider may have returned an
	// errInfo response themselves.
	errResponse, err := GetSuccessPayloadResponse[oauth.DeviceAuthStatusErrorInfo](w.Body)
	assert.NoError(t, err)

	assert.Equal(t, 403, errResponse.StatusCode)
	assert.Equal(t, "got an error", errResponse.ErrorCode)
	assert.Equal(t, "error message", errResponse.ErrorMessage)
	assert.Equal(t, "https://error-info-here.com", errResponse.ErrorURI)
}

func setupAPIListenerForPlusOAuth(t *testing.T, oauthConfig *oauth.Config, username string) (al *APIListener, mockUsersService *MockUsersService) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	plusConfig := &rportplus.PlusConfig{
		PluginConfig: &rportplus.PluginConfig{
			PluginPath: defaultPluginPath,
		},
		OAuthConfig: oauthConfig,
	}

	plusManager := &plusManagerForMockOAuth{}
	plusManager.InitPlusManager(plusConfig, nil, plusLog)

	_, err := plusManager.RegisterCapability(plusMockOAuthCapability, &oauthmock.Capability{
		Config: oauthConfig,
		Logger: plusLog,

		Provider: &oauthmock.MockCapabilityProvider{
			Username: username,
		},
	})
	require.NoError(t, err)

	al, mockUsersService = setupTestAPIListenerForOAuth(t, plusManager, plusConfig, oauthConfig)

	return al, mockUsersService
}

func setupTestAPIListenerForOAuth(
	t *testing.T,
	plusManager rportplus.Manager,
	plusConfig *rportplus.PlusConfig,
	oauthConfig *oauth.Config,
) (al *APIListener, mockUsersService *MockUsersService) {
	userGroup := "Administrators"
	user := &users.User{
		Username: "user1",
		Password: "$2y$05$ep2DdPDeLDDhwRrED9q/vuVEzRpZtB5WHCFT7YbcmH9r9oNmlsZOm",
	}
	mockUsersService = &MockUsersService{
		UserService: users.NewAPIService(users.NewStaticProvider([]*users.User{user}), false, 0, -1),
	}
	mockTokenManager := authorization.NewManager(
		CommonAPITokenTestDb(t, "user1", "prefixtkn", "the name", authorization.APITokenReadWrite, "mynicefi-xedl-enth-long-livedpasswor")) // APIToken database

	plusConfig.OAuthConfig = oauthConfig

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

	return al, mockUsersService
}
