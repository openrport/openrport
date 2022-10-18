package chserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rportplus "github.com/cloudradar-monitoring/rport/plus"
	"github.com/cloudradar-monitoring/rport/plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/plus/capabilities/oauthmock"
	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/ptr"
	"github.com/cloudradar-monitoring/rport/share/security"
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

	config := &Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: &rportplus.PlusConfig{
			PluginPath: defaultPluginPath,
		},
		OAuthConfig: &oauth.Config{
			Provider: oauth.GitHubOAuthProvider,
		},
	}

	plus = &plusManagerForMockOAuth{}
	plus.InitPlusManager(config.PlusConfig, plusLog)
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
	al, _ := SetupAPIListener(t,
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
		PluginPath: defaultPluginPath,
	}

	oauthConfig := &oauth.Config{
		Provider: oauth.GitHubOAuthProvider,
	}

	plusManager := initMockPlusManager()

	_, err := plusManager.RegisterCapability(plusMockOAuthCapability, &oauthmock.Capability{
		Config: oauthConfig,
		Logger: plusLog,
	})
	require.NoError(t, err)

	al, _ := setupTestAPIListenerForOAuth(t, plusManager, plusConfig, oauthConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+authRoutesPrefix+authProviderRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	info := &SuccessPayloadResponse[AuthProviderInfo]{}
	err = GetSuccessPayloadResponse(w.Body, info)
	assert.NoError(t, err)

	assert.Equal(t, "github", info.Data.AuthProvider)
	assert.Equal(t, allRoutesPrefix+authRoutesPrefix+authSettingsRoute, info.Data.SettingsURI)
	assert.Equal(t, allRoutesPrefix+authRoutesPrefix+authDeviceSettingsRoute, info.Data.DeviceSettingsURI)
}

type AuthSettingsResponse struct {
	Data AuthSettings
}

func setupPlusOAuth() (plusManager rportplus.Manager, plusConfig *rportplus.PlusConfig, oauthConfig *oauth.Config, plusLog *logger.Logger) {
	plusLog = logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	plusConfig = &rportplus.PlusConfig{
		PluginPath: defaultPluginPath,
	}

	oauthConfig = &oauth.Config{
		Provider: oauth.GitHubOAuthProvider,
	}

	plusManager = &plusManagerForMockOAuth{}
	plusManager.InitPlusManager(plusConfig, plusLog)

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
	req := httptest.NewRequest("GET", "/api/v1"+authRoutesPrefix+authSettingsRoute, nil)

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
	req := httptest.NewRequest("GET", "/api/v1"+authRoutesPrefix+authDeviceSettingsRoute, nil)

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
	req := httptest.NewRequest("GET", "/api/v1"+authRoutesPrefix+authDeviceSettingsRoute, nil)

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
	req := httptest.NewRequest("GET", "/api/v1"+authRoutesPrefix+authSettingsRoute, nil)

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
			al, mockUsersService := SetupAPIListener(t, tc.OAuthConfig, tc.Username)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", allRoutesPrefix+oauth.DefaultLoginURI, nil)

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
	errResponse := &SuccessPayloadResponse[oauth.DeviceAuthStatusErrorInfo]{}
	err = GetSuccessPayloadResponse(w.Body, errResponse)
	assert.NoError(t, err)

	assert.Equal(t, 403, errResponse.Data.StatusCode)
	assert.Equal(t, "got an error", errResponse.Data.ErrorCode)
	assert.Equal(t, "error message", errResponse.Data.ErrorMessage)
	assert.Equal(t, "https://error-info-here.com", errResponse.Data.ErrorURI)
}

func SetupAPIListener(t *testing.T, oauthConfig *oauth.Config, username string) (al *APIListener, mockUsersService *MockUsersService) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	plusConfig := &rportplus.PlusConfig{
		PluginPath: defaultPluginPath,
	}

	plusManager := &plusManagerForMockOAuth{}
	plusManager.InitPlusManager(plusConfig, plusLog)

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
		Token:    ptr.String("$2y$05$/D7g/d0sDkNSOh.e6Jzc9OWClcpZ1ieE8Dx.WUaWgayd3Ab0rRdxu"),
	}
	mockUsersService = &MockUsersService{
		UserService: users.NewAPIService(users.NewStaticProvider([]*users.User{user}), false),
	}

	al = &APIListener{
		Server: &Server{
			config: &Config{
				API: APIConfig{
					DefaultUserGroup: userGroup,
				},
				PlusConfig:  plusConfig,
				OAuthConfig: oauthConfig,
			},
			plusManager: plusManager,
		},
		bannedUsers: security.NewBanList(0),
		userService: mockUsersService,
		apiSessions: newEmptyAPISessionCache(t),
	}
	al.initRouter()

	return al, mockUsersService
}
