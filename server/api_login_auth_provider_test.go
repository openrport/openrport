package chserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rportplus "github.com/cloudradar-monitoring/rport/plus"
	"github.com/cloudradar-monitoring/rport/plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/security"
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
			config: &Config{
				API:        APIConfig{},
				PlusConfig: rportplus.PlusConfig{},
			},
			plusManager: nil,
		},
		bannedUsers: security.NewBanList(0),
		apiSessions: newEmptyAPISessionCache(t),
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+authRoutesPrefix+authProviderRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	info, err := GetSuccessPayloadResponse[AuthProviderInfo](w.Body)
	assert.NoError(t, err)

	assert.Equal(t, BuiltInAuthProviderName, info.AuthProvider)
	assert.Equal(t, "", info.SettingsURI)
}

func TestHandleGetAuthSettingsWhenNoPlusOAuth(t *testing.T) {
	al := APIListener{
		Server: &Server{
			config: &Config{
				API:        APIConfig{},
				PlusConfig: rportplus.PlusConfig{},
			},
			plusManager: nil,
		},
		bannedUsers: security.NewBanList(0),
		apiSessions: newEmptyAPISessionCache(t),
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1"+authRoutesPrefix+authSettingsRoute, nil)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
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

	plusConfig := rportplus.PlusConfig{
		PluginConfig: &rportplus.PluginConfig{
			PluginPath: defaultPluginPath,
		},
		OAuthConfig: oauthConfig,
	}

	filesAPI := files.NewFileSystem()

	plusManager, err := rportplus.NewPlusManager(&plusConfig, plusLog, filesAPI)
	if err != nil {
		t.Skipf("plus plugin not available: %s", err)
	}

	serverCfg := &Config{
		API:        APIConfig{},
		PlusConfig: plusConfig,
	}

	err = RegisterPlusCapabilities(plusManager, serverCfg, plusLog)
	require.NoError(t, err)

	al := APIListener{
		Server: &Server{
			config: &Config{
				API:        APIConfig{},
				PlusConfig: plusConfig,
			},
			plusManager: plusManager,
		},
		bannedUsers: security.NewBanList(0),
		apiSessions: newEmptyAPISessionCache(t),
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", allRoutesPrefix+authRoutesPrefix+authSettingsRoute, nil)

	al.router.ServeHTTP(w, req)

	var settings AuthSettingsResponse
	err = json.NewDecoder(w.Body).Decode(&settings)
	assert.NoError(t, err)

	loginInfo := settings.Data.LoginInfo

	assert.NotEmpty(t, loginInfo.AuthorizeURL)
	assert.Equal(t, allRoutesPrefix+oauth.DefaultLoginURI, loginInfo.LoginURI)
	assert.NotEmpty(t, loginInfo.State)
	assert.NotEmpty(t, loginInfo.Expiry)
	assert.Equal(t, http.StatusOK, w.Code)
}
