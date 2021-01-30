package chclient

import (
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

var defaultValidMinConfig = Config{
	Client: ClientConfig{
		Server: "test.com",
	},
	RemoteCommands: CommandsConfig{
		Enabled:       true,
		SendBackLimit: 2048,
		Order:         allowDenyOrder,
		allowRegexp:   []*regexp.Regexp{regexp.MustCompile(".*")},
	},
}

func TestConfigParseAndValidateHeaders(t *testing.T) {
	testCases := []struct {
		Name           string
		ConnConfig     ConnectionConfig
		ExpectedHeader http.Header
	}{
		{
			Name: "defaults",
			ExpectedHeader: http.Header{
				"User-Agent": []string{"rport 0.0.0-src"},
			},
		}, {
			Name: "host set",
			ConnConfig: ConnectionConfig{
				Hostname: "test.com",
			},
			ExpectedHeader: http.Header{
				"Host":       []string{"test.com"},
				"User-Agent": []string{"rport 0.0.0-src"},
			},
		}, {
			Name: "user agent set in config",
			ConnConfig: ConnectionConfig{
				HeadersRaw: []string{"User-Agent: test-agent"},
			},
			ExpectedHeader: http.Header{
				"User-Agent": []string{"test-agent"},
			},
		}, {
			Name: "multiple headers set",
			ConnConfig: ConnectionConfig{
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
			config := defaultValidMinConfig
			config.Connection = tc.ConnConfig

			err := config.ParseAndValidate()
			require.NoError(t, err)

			assert.Equal(t, tc.ExpectedHeader, config.Connection.Headers())
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
			config := defaultValidMinConfig
			config.Client.Server = tc.ServerURL

			err := config.ParseAndValidate()

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
			config := defaultValidMinConfig
			config.Connection.MaxRetryInterval = tc.MaxRetryInterval
			err := config.ParseAndValidate()

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
			config := defaultValidMinConfig
			config.Client.Proxy = tc.Proxy
			err := config.ParseAndValidate()

			if tc.ExpectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.ExpectedProxyURL, config.Client.proxyURL)
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
		ExpectedRemotes []*chshare.Remote
		ExpectedError   string
	}{
		{
			Name:            "not set",
			Remotes:         []string{},
			ExpectedRemotes: []*chshare.Remote{},
		}, {
			Name:    "one",
			Remotes: []string{"8000"},
			ExpectedRemotes: []*chshare.Remote{
				&chshare.Remote{
					RemoteHost: "0.0.0.0",
					RemotePort: "8000",
				},
			},
		}, {
			Name:    "multiple",
			Remotes: []string{"8000", "3000"},
			ExpectedRemotes: []*chshare.Remote{
				&chshare.Remote{
					RemoteHost: "0.0.0.0",
					RemotePort: "8000",
				},
				&chshare.Remote{
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
			config := defaultValidMinConfig
			config.Client.Remotes = tc.Remotes
			err := config.ParseAndValidate()

			if tc.ExpectedError == "" {
				require.NoError(t, err)
				assert.ElementsMatch(t, tc.ExpectedRemotes, config.Client.remotes)
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
			config := defaultValidMinConfig
			config.Client.Auth = tc.Auth
			err := config.ParseAndValidate()

			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedUser, config.Client.authUser)
			assert.Equal(t, tc.ExpectedPass, config.Client.authPass)
		})
	}
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
			config := defaultValidMinConfig
			config.RemoteCommands.SendBackLimit = tc.sendBackLimit

			// when
			gotErr := config.ParseAndValidate()

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
			config := defaultValidMinConfig
			config.RemoteCommands.Allow = tc.allow

			// when
			gotErr := config.ParseAndValidate()

			// then
			if tc.wantErrContains != "" {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, gotErr)
				assert.ElementsMatch(t, tc.allow, convertToRegexpStrList(config.RemoteCommands.allowRegexp))
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
			config := defaultValidMinConfig
			config.RemoteCommands.Deny = tc.deny

			// when
			gotErr := config.ParseAndValidate()

			// then
			if tc.wantErrContains != "" {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, gotErr)
				assert.ElementsMatch(t, tc.deny, convertToRegexpStrList(config.RemoteCommands.denyRegexp))
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
			config := defaultValidMinConfig
			config.RemoteCommands.Order = tc.order

			// when
			gotErr := config.ParseAndValidate()

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
