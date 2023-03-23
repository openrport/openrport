package chserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rportplus "github.com/realvnc-labs/rport/plus"
	"github.com/realvnc-labs/rport/plus/capabilities/oauth"
	"github.com/realvnc-labs/rport/plus/license"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/server/routes"
	"github.com/realvnc-labs/rport/share/files"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/security"
)

// PayloadResponse represents the types that might be returned by a successful api response
type PayloadResponse interface {
	AuthProviderInfo | oauth.DeviceAuthStatusErrorInfo
}

// SuccessPayloadResponse adds the "data" (in json) structural element of a successful api
// response
type SuccessPayloadResponse[T PayloadResponse] struct {
	Data *T
}

// GetSuccessPayloadResponse returns a successful payload response of the expected type
func GetSuccessPayloadResponse[R PayloadResponse](r io.Reader) (response *R, err error) {
	resp := &SuccessPayloadResponse[R]{}
	err = json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func TestHandleGetBuiltInAuthProvider(t *testing.T) {
	al := APIListener{
		Server: &Server{
			config: &chconfig.Config{
				API: chconfig.APIConfig{
					MaxTokenLifeTimeHours: 999,
				},
				PlusConfig: rportplus.PlusConfig{},
			},
			plusManager: nil,
		},
		bannedUsers: security.NewBanList(0),
		apiSessions: newEmptyAPISessionCache(t),
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AuthRoutesPrefix+routes.AuthProviderRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	info, err := GetSuccessPayloadResponse[AuthProviderInfo](w.Body)
	assert.NoError(t, err)

	assert.Equal(t, BuiltInAuthProviderName, info.AuthProvider)
	assert.Equal(t, "", info.SettingsURI)
	assert.Equal(t, 999, info.MaxTokenLifetime)
}

func TestHandleGetAuthSettingsWhenNoPlusOAuth(t *testing.T) {
	al := APIListener{
		Server: &Server{
			config: &chconfig.Config{
				API:        chconfig.APIConfig{},
				PlusConfig: rportplus.PlusConfig{},
			},
			plusManager: nil,
		},
		bannedUsers: security.NewBanList(0),
		apiSessions: newEmptyAPISessionCache(t),
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AuthRoutesPrefix+routes.AuthSettingsRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleGetAuthProviderWhenPlusOAuthAvailable(t *testing.T) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	oauthConfig := &oauth.Config{
		Provider:             oauth.GitHubOAuthProvider,
		BaseAuthorizeURL:     "https://test.com/authorize",
		TokenURL:             "https://test.com/access_token",
		RedirectURI:          "https://test.com/callback",
		ClientID:             "1234567890",
		ClientSecret:         "0987654321",
		RequiredOrganization: "testorg",
	}

	// note: the license actually isn't given time to verify, which is why test works
	licConfig := &license.Config{
		ID:      "83c5afc7-87a7-4a3d-9889-3905ec979045",
		Key:     "6OO1STn0b0XUahz+RN6jBJ93KBuSbsKPef+SMl98NEU=",
		DataDir: ".",
	}

	plusConfig := rportplus.PlusConfig{
		PluginConfig: &rportplus.PluginConfig{
			PluginPath: defaultPluginPath,
		},
		LicenseConfig: licConfig,
		OAuthConfig:   oauthConfig,
	}

	filesAPI := files.NewFileSystem()

	ctx := context.Background()

	plusManager, err := rportplus.NewPlusManager(ctx, &plusConfig, nil, plusLog, filesAPI)
	if err != nil {
		t.Skipf("plus plugin not available: %s", err)
	}

	serverCfg := &chconfig.Config{
		API: chconfig.APIConfig{
			MaxTokenLifeTimeHours: 999,
		},
		PlusConfig: plusConfig,
	}

	err = RegisterPlusCapabilities(plusManager, serverCfg, plusLog)
	require.NoError(t, err)

	al := APIListener{
		Server: &Server{
			config:      serverCfg,
			plusManager: nil,
		},
		bannedUsers: security.NewBanList(0),
		apiSessions: newEmptyAPISessionCache(t),
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AuthRoutesPrefix+routes.AuthProviderRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	info, err := GetSuccessPayloadResponse[AuthProviderInfo](w.Body)
	assert.NoError(t, err)

	assert.Equal(t, oauth.GitHubOAuthProvider, info.AuthProvider)
	assert.Equal(t, routes.AllRoutesPrefix+routes.AuthRoutesPrefix+routes.AuthSettingsRoute, info.SettingsURI)
	assert.Equal(t, 999, info.MaxTokenLifetime)
}

func TestHandleGetAuthSettingsWhenPlusOAuthAvailable(t *testing.T) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	oauthConfig := &oauth.Config{
		Provider:             oauth.GitHubOAuthProvider,
		BaseAuthorizeURL:     "https://test.com/authorize",
		TokenURL:             "https://test.com/access_token",
		RedirectURI:          "https://test.com/callback",
		ClientID:             "1234567890",
		ClientSecret:         "0987654321",
		RequiredOrganization: "testorg",
	}

	// note: the license actually isn't given time to verify, which is why test works
	licConfig := &license.Config{
		ID:      "83c5afc7-87a7-4a3d-9889-3905ec979045",
		Key:     "6OO1STn0b0XUahz+RN6jBJ93KBuSbsKPef+SMl98NEU=",
		DataDir: ".",
	}

	plusConfig := rportplus.PlusConfig{
		PluginConfig: &rportplus.PluginConfig{
			PluginPath: defaultPluginPath,
		},
		LicenseConfig: licConfig,
		OAuthConfig:   oauthConfig,
	}

	filesAPI := files.NewFileSystem()

	ctx := context.Background()

	plusManager, err := rportplus.NewPlusManager(ctx, &plusConfig, nil, plusLog, filesAPI)
	if err != nil {
		t.Skipf("plus plugin not available: %s", err)
	}

	serverCfg := &chconfig.Config{
		API:        chconfig.APIConfig{},
		PlusConfig: plusConfig,
	}

	err = RegisterPlusCapabilities(plusManager, serverCfg, plusLog)
	require.NoError(t, err)

	al := APIListener{
		Server: &Server{
			config:      serverCfg,
			plusManager: plusManager,
		},
		bannedUsers: security.NewBanList(0),
		apiSessions: newEmptyAPISessionCache(t),
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", routes.AllRoutesPrefix+routes.AuthRoutesPrefix+routes.AuthSettingsRoute, nil)

	al.router.ServeHTTP(w, req)

	var settings AuthSettingsResponse
	err = json.NewDecoder(w.Body).Decode(&settings)
	assert.NoError(t, err)

	loginInfo := settings.Data.LoginInfo

	assert.NotEmpty(t, loginInfo.AuthorizeURL)
	assert.Equal(t, routes.AllRoutesPrefix+oauth.DefaultLoginURI, loginInfo.LoginURI)
	assert.NotEmpty(t, loginInfo.State)
	assert.NotEmpty(t, loginInfo.Expiry)
	assert.Equal(t, http.StatusOK, w.Code)
}
