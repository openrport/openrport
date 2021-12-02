package chclient

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/share/clientconfig"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func getDefaultValidMinConfig() ClientConfigHolder {
	return ClientConfigHolder{
		Config: &clientconfig.Config{
			Client: clientconfig.ClientConfig{
				Server:  "test.com",
				DataDir: os.TempDir(),
			},
			RemoteCommands: clientconfig.CommandsConfig{
				Enabled:       true,
				SendBackLimit: 2048,
				Order:         allowDenyOrder,
				AllowRegexp:   []*regexp.Regexp{regexp.MustCompile(".*")},
			},
			RemoteScripts: clientconfig.ScriptsConfig{
				Enabled: false,
			},
		},
	}
}

func TestConfigParseAndValidateHeaders(t *testing.T) {
	testCases := []struct {
		Name           string
		ConnConfig     clientconfig.ConnectionConfig
		ExpectedHeader http.Header
	}{
		{
			Name: "defaults",
			ExpectedHeader: http.Header{
				"User-Agent": []string{"rport 0.0.0-src"},
			},
		}, {
			Name: "host set",
			ConnConfig: clientconfig.ConnectionConfig{
				Hostname: "test.com",
			},
			ExpectedHeader: http.Header{
				"Host":       []string{"test.com"},
				"User-Agent": []string{"rport 0.0.0-src"},
			},
		}, {
			Name: "user agent set in config",
			ConnConfig: clientconfig.ConnectionConfig{
				HeadersRaw: []string{"User-Agent: test-agent"},
			},
			ExpectedHeader: http.Header{
				"User-Agent": []string{"test-agent"},
			},
		}, {
			Name: "multiple headers set",
			ConnConfig: clientconfig.ConnectionConfig{
				HeadersRaw: []string{"Test1: v1", "Test2: v2"},
			},
			ExpectedHeader: http.Header{
				"Test1":      []string{"v1"},
				"Test2":      []string{"v2"},
				"User-Agent": []string{"rport 0.0.0-src"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			config := getDefaultValidMinConfig()
			config.Connection = tc.ConnConfig

			err := config.ParseAndValidate(true)
			require.NoError(t, err)

			assert.Equal(t, tc.ExpectedHeader, config.Connection.HTTPHeaders)
		})
	}
}

func TestConfigParseAndValidateServerURL(t *testing.T) {
	testCases := []struct {
		ServerURL     string
		ExpectedURL   string
		ExpectedError string
	}{
		{
			ServerURL:     "",
			ExpectedError: "server address is required",
		}, {
			ServerURL:   "test.com",
			ExpectedURL: "ws://test.com:80",
		}, {
			ServerURL:   "http://test.com",
			ExpectedURL: "ws://test.com:80",
		}, {
			ServerURL:   "https://test.com",
			ExpectedURL: "wss://test.com:443",
		}, {
			ServerURL:   "http://test.com:1234",
			ExpectedURL: "ws://test.com:1234",
		}, {
			ServerURL:   "https://test.com:1234",
			ExpectedURL: "wss://test.com:1234",
		}, {
			ServerURL:   "ws://test.com:1234",
			ExpectedURL: "ws://test.com:1234",
		}, {
			ServerURL:   "wss://test.com:1234",
			ExpectedURL: "wss://test.com:1234",
		}, {
			ServerURL:     "test\n.com",
			ExpectedError: `invalid server address: parse "http://test\n.com": net/url: invalid control character in URL`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.ServerURL, func(t *testing.T) {
			config := getDefaultValidMinConfig()
			config.Client.Server = tc.ServerURL

			err := config.ParseAndValidate(true)

			if tc.ExpectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.ExpectedURL, config.Client.Server)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.ExpectedError, err.Error())
			}
		})
	}
}

func TestConfigParseAndValidateMaxRetryInterval(t *testing.T) {
	testCases := []struct {
		Name                     string
		MaxRetryInterval         time.Duration
		ExpectedMaxRetryInterval time.Duration
	}{
		{
			Name:                     "minimum max retry interval",
			MaxRetryInterval:         time.Second,
			ExpectedMaxRetryInterval: time.Second,
		}, {
			Name:                     "set max retry interval",
			MaxRetryInterval:         time.Minute,
			ExpectedMaxRetryInterval: time.Minute,
		}, {
			Name:                     "default",
			MaxRetryInterval:         time.Duration(0),
			ExpectedMaxRetryInterval: 5 * time.Minute,
		}, {
			Name:                     "small retry interval",
			MaxRetryInterval:         time.Millisecond,
			ExpectedMaxRetryInterval: 5 * time.Minute,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			config := getDefaultValidMinConfig()
			config.Connection.MaxRetryInterval = tc.MaxRetryInterval
			err := config.ParseAndValidate(true)

			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedMaxRetryInterval, config.Connection.MaxRetryInterval)
		})
	}
}

func TestConfigParseAndValidateProxyURL(t *testing.T) {
	expectedProxyURL, err := url.Parse("http://proxy.com")
	require.NoError(t, err)

	testCases := []struct {
		Name             string
		Proxy            string
		ExpectedProxyURL *url.URL
		ExpectedError    string
	}{
		{
			Name:             "not set",
			Proxy:            "",
			ExpectedProxyURL: nil,
		}, {
			Name:          "invalid",
			Proxy:         "http://proxy\n.com",
			ExpectedError: `invalid proxy URL: parse "http://proxy\n.com": net/url: invalid control character in URL`,
		}, {
			Name:             "with proxy",
			Proxy:            "http://proxy.com",
			ExpectedProxyURL: expectedProxyURL,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			config := getDefaultValidMinConfig()
			config.Client.Proxy = tc.Proxy
			err := config.ParseAndValidate(true)

			if tc.ExpectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.ExpectedProxyURL, config.Client.ProxyURL)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.ExpectedError, err.Error())
			}
		})
	}
}

func TestConfigParseAndValidateRemotes(t *testing.T) {
	testCases := []struct {
		Name            string
		Remotes         []string
		ExpectedRemotes []*models.Remote
		ExpectedError   string
	}{
		{
			Name:            "not set",
			Remotes:         []string{},
			ExpectedRemotes: []*models.Remote{},
		}, {
			Name:    "one",
			Remotes: []string{"8000"},
			ExpectedRemotes: []*models.Remote{
				&models.Remote{
					RemoteHost: "0.0.0.0",
					RemotePort: "8000",
				},
			},
		}, {
			Name:    "multiple",
			Remotes: []string{"8000", "3000"},
			ExpectedRemotes: []*models.Remote{
				&models.Remote{
					RemoteHost: "0.0.0.0",
					RemotePort: "8000",
				},
				&models.Remote{
					RemoteHost: "0.0.0.0",
					RemotePort: "3000",
				},
			},
		}, {
			Name:          "invalid",
			Remotes:       []string{"abc"},
			ExpectedError: `failed to decode remote "abc": Missing ports`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			config := getDefaultValidMinConfig()
			config.Client.Remotes = tc.Remotes
			err := config.ParseAndValidate(true)

			if tc.ExpectedError == "" {
				require.NoError(t, err)
				assert.ElementsMatch(t, tc.ExpectedRemotes, config.Client.Tunnels)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.ExpectedError, err.Error())
			}
		})
	}
}

func TestConfigParseAndValidateAuth(t *testing.T) {
	testCases := []struct {
		Auth         string
		ExpectedUser string
		ExpectedPass string
	}{
		{
			Auth:         "",
			ExpectedUser: "",
			ExpectedPass: "",
		}, {
			Auth:         "test:pass123",
			ExpectedUser: "test",
			ExpectedPass: "pass123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Auth, func(t *testing.T) {
			config := getDefaultValidMinConfig()
			config.Client.Auth = tc.Auth
			err := config.ParseAndValidate(true)

			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedUser, config.Client.AuthUser)
			assert.Equal(t, tc.ExpectedPass, config.Client.AuthPass)
		})
	}
}

func TestScriptsExecutionEnabledButCommandsDisabled(t *testing.T) {
	config := getDefaultValidMinConfig()
	config.RemoteScripts.Enabled = true
	config.RemoteCommands.Enabled = false
	err := config.ParseAndValidate(false)

	require.EqualError(t, err, "remote scripts execution requires remote commands to be enabled")

	err1 := config.ParseAndValidate(true)
	require.NoError(t, err1)
}

func TestConfigParseAndValidateSendBackLimit(t *testing.T) {
	testCases := []struct {
		name            string
		sendBackLimit   int
		wantErrContains string
	}{
		{
			name:            "zero limit",
			sendBackLimit:   0,
			wantErrContains: "",
		},
		{
			name:            "valid positive limit",
			sendBackLimit:   1,
			wantErrContains: "",
		},
		{
			name:            "invalid limit negative",
			sendBackLimit:   -1,
			wantErrContains: "send back limit can not be negative",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			config := getDefaultValidMinConfig()
			config.RemoteCommands.SendBackLimit = tc.sendBackLimit

			// when
			gotErr := config.ParseAndValidate(true)

			// then
			if tc.wantErrContains != "" {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, gotErr)
			}
		})
	}
}

func TestConfigParseAndValidateAllowRegexp(t *testing.T) {
	testCases := []struct {
		name            string
		allow           []string
		wantErrContains string
	}{
		{
			name:  "unset",
			allow: nil,
		},
		{
			name:  "empty",
			allow: []string{},
		},
		{
			name:  "valid",
			allow: []string{"^/usr/bin.*", "^/usr/local/bin/.*", `^C:\\Windows\\System32.*`},
		},
		{
			name:            "invalid",
			allow:           []string{"^/usr/bin.*", "{invalid regexp)", `^C:\\Windows\\System32.*`},
			wantErrContains: "allow regexp: invalid regular expression",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			config := getDefaultValidMinConfig()
			config.RemoteCommands.Allow = tc.allow

			// when
			gotErr := config.ParseAndValidate(true)

			// then
			if tc.wantErrContains != "" {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, gotErr)
				assert.ElementsMatch(t, tc.allow, convertToRegexpStrList(config.RemoteCommands.AllowRegexp))
			}
		})
	}
}

func TestConfigParseAndValidateDenyRegexp(t *testing.T) {
	testCases := []struct {
		name            string
		deny            []string
		wantErrContains string
	}{
		{
			name: "unset",
			deny: nil,
		},
		{
			name: "empty",
			deny: []string{},
		},
		{
			name: "valid",
			deny: []string{"^/usr/bin/zip.*", `^C:\\Windows\\.*`},
		},
		{
			name:            "invalid",
			deny:            []string{"^/usr/bin/zip.*", "{invalid regexp)", `^C:\\Windows\\.*`},
			wantErrContains: "deny regexp: invalid regular expression",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			config := getDefaultValidMinConfig()
			config.RemoteCommands.Deny = tc.deny

			// when
			gotErr := config.ParseAndValidate(true)

			// then
			if tc.wantErrContains != "" {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, gotErr)
				assert.ElementsMatch(t, tc.deny, convertToRegexpStrList(config.RemoteCommands.DenyRegexp))
			}
		})
	}
}

func convertToRegexpStrList(regexpList []*regexp.Regexp) []string {
	var res []string
	for _, r := range regexpList {
		res = append(res, r.String())
	}
	return res
}

func TestConfigParseAndValidateAllowDenyOrder(t *testing.T) {
	testCases := []struct {
		name            string
		order           [2]string
		wantErrContains string
	}{
		{
			name:  "valid: allow deny",
			order: allowDenyOrder,
		},
		{
			name:  "valid: deny allow",
			order: allowDenyOrder,
		},
		{
			name:            "invalid: empty",
			order:           [2]string{},
			wantErrContains: "invalid order:",
		},
		{
			name:            "invalid value",
			order:           [2]string{"deny", "unknown"},
			wantErrContains: "invalid order:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			config := getDefaultValidMinConfig()
			config.RemoteCommands.Order = tc.order

			// when
			gotErr := config.ParseAndValidate(true)

			// then
			if tc.wantErrContains != "" {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, gotErr)
			}
		})
	}
}

func TestConfigParseAndValidateFallbackServers(t *testing.T) {
	testCases := []struct {
		Name            string
		FallbackServers []string
		Expected        []string
		ExpectedError   error
	}{
		{
			Name:            "No fallback servers is ok",
			FallbackServers: nil,
			ExpectedError:   nil,
		}, {
			Name:            "No protocol",
			FallbackServers: []string{"test.com"},
			Expected:        []string{"ws://test.com:80"},
		}, {
			Name:            "http",
			FallbackServers: []string{"http://test.com"},
			Expected:        []string{"ws://test.com:80"},
		}, {
			Name:            "https",
			FallbackServers: []string{"https://test.com"},
			Expected:        []string{"wss://test.com:443"},
		}, {
			Name:            "ws",
			FallbackServers: []string{"ws://test.com"},
			Expected:        []string{"ws://test.com:80"},
		}, {
			Name:            "wss",
			FallbackServers: []string{"wss://test.com"},
			Expected:        []string{"wss://test.com:443"},
		}, {
			Name:            "Custom port",
			FallbackServers: []string{"http://test.com:1234"},
			Expected:        []string{"ws://test.com:1234"},
		}, {
			Name:            "Multiple",
			FallbackServers: []string{"http://test.com:1234", "example.com"},
			Expected:        []string{"ws://test.com:1234", "ws://example.com:80"},
		}, {
			Name:            "Invalid url",
			FallbackServers: []string{"test\n.com"},
			ExpectedError:   errors.New(`invalid fallback server address: parse "http://test\n.com": net/url: invalid control character in URL`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			config := getDefaultValidMinConfig()
			config.Client.FallbackServers = tc.FallbackServers

			err := config.ParseAndValidate(true)

			assert.Equal(t, tc.ExpectedError, err)
			if tc.ExpectedError == nil {
				assert.Equal(t, tc.Expected, config.Client.FallbackServers)
			}
		})
	}
}
