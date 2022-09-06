package chserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rportplus "github.com/cloudradar-monitoring/rport/rport-plus"
	"github.com/cloudradar-monitoring/rport/rport-plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/rport-plus/capabilities/oauthmock"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/ptr"
	"github.com/cloudradar-monitoring/rport/share/security"
)

const (
	PlusMockOAuthCapability = "plus-oauth-mock"
)

type plusManagerForMockOAuth struct {
	rportplus.ManagerProvider
}

// GetOAuthCapabilityEx overrides the default implementation to return a mock oauth
// capability provider
func (pm *plusManagerForMockOAuth) GetOAuthCapabilityEx() (capEx oauth.CapabilityEx) {
	capEntry := pm.ManagerProvider.GetCapability(PlusMockOAuthCapability)
	cap := capEntry.(*oauthmock.Capability)
	capEx = cap.GetOAuthCapabilityEx()
	return capEx
}

func TestHandleGetOAuthLoginURLs(t *testing.T) {
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

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var loginResponse LoginInfoResponse

	err := json.NewDecoder(w.Body).Decode(&loginResponse)
	assert.NoError(t, err)

	loginErrors := loginResponse.Errors
	loginError := loginErrors[0]

	assert.Equal(t, "mock login msg", loginError.Detail)
	assert.Equal(t, "mock login url", loginError.LoginURL)
	assert.Equal(t, "/api/v1/plus/mock_exchange_uri", loginError.ExchangeURI)
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
			req := httptest.NewRequest("GET", "/api/v1/plus/oauth/exchangecode", nil)

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

type OAuthConfigResponse struct {
	Data *oauth.Config `json:"data"`
}

func TestShouldProvidePlusOauthStatus(t *testing.T) {
	al, _ := SetupAPIListener(t,
		&oauth.Config{
			Provider:          oauth.GitHubOAuthProvider,
			PermittedUserList: true,
			ClientSecret:      "1234",
		},
		"user1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/plus/oauth/status", nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var cfgResponse OAuthConfigResponse

	err := json.NewDecoder(w.Body).Decode(&cfgResponse)
	assert.NoError(t, err)

	cfg := cfgResponse.Data
	assert.Equal(t, oauth.GitHubOAuthProvider, cfg.Provider)
	assert.Equal(t, true, cfg.PermittedUserList)
	// ensure not returned by the server
	assert.Equal(t, "", cfg.ClientSecret)
}

func SetupAPIListener(t *testing.T, oauthConfig *oauth.Config, username string) (al *APIListener, mockUsersService *MockUsersService) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	plusConfig := &rportplus.PlusConfig{
		PluginPath: defaultPluginPath,
	}

	plusManager := &plusManagerForMockOAuth{}
	plusManager.InitPlusManager(plusConfig, plusLog)

	_, err := plusManager.RegisterCapability(PlusMockOAuthCapability, &oauthmock.Capability{
		Config: oauthConfig,
		Logger: plusLog,

		Provider: &oauthmock.MockCapabilityProvider{
			Username: username,
		},
	})
	require.NoError(t, err)

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
