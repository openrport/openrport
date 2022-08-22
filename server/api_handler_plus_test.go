package chserver

import (
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

func TestHandleOAuthAuthorizationCode(t *testing.T) {
	testCases := []struct {
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
				CreateMissingUsers:   false,
			},
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name: "create missing user",
			OAuthConfig: &oauth.Config{
				Provider:             oauth.GitHubOAuthProvider,
				RequiredOrganization: "cloudradar",
				CreateMissingUsers:   true,
			},
			Username:       "added-user",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name: "use auth file with known user",
			OAuthConfig: &oauth.Config{
				Provider:           oauth.GitHubOAuthProvider,
				UseAuthFile:        true,
				CreateMissingUsers: false,
			},
			Username:       "user1",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name: "use auth file with unknown user",
			OAuthConfig: &oauth.Config{
				Provider:           oauth.GitHubOAuthProvider,
				UseAuthFile:        true,
				CreateMissingUsers: false,
			},
			Username:       "unknown-user",
			ExpectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			al, mockUsersService := SetupAPIListener(t, tc.OAuthConfig, tc.Username)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/oauth/callback", nil)

			al.router.ServeHTTP(w, req)

			assert.Equal(t, tc.ExpectedStatus, w.Code)

			if w.Code == http.StatusOK {
				if tc.ExpectedStatus == http.StatusOK {
					assert.Contains(t, w.Body.String(), `{"data":{"token":"`)
				}
			}

			if tc.OAuthConfig.CreateMissingUsers {
				changedUser := mockUsersService.ChangeUser
				assert.Equal(t, tc.Username, changedUser.Username)
			}
		})
	}

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
