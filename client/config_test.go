package chclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigParseAndValidateHeaders(t *testing.T) {
	testCases := []struct {
		Name           string
		Config         Config
		ExpectedHeader http.Header
	}{
		{
			Name: "defaults",
			ExpectedHeader: http.Header{
				"User-Agent": []string{"rport 0.0.0-src"},
			},
		}, {
			Name: "host set",
			Config: Config{
				Connection: ConnectionConfig{
					Hostname: "test.com",
				},
			},
			ExpectedHeader: http.Header{
				"Host":       []string{"test.com"},
				"User-Agent": []string{"rport 0.0.0-src"},
			},
		}, {
			Name: "user agent set in config",
			Config: Config{
				Connection: ConnectionConfig{
					HeadersRaw: []string{"User-Agent: test-agent"},
				},
			},
			ExpectedHeader: http.Header{
				"User-Agent": []string{"test-agent"},
			},
		}, {
			Name: "multiple headers set",
			Config: Config{
				Connection: ConnectionConfig{
					HeadersRaw: []string{"Test1: v1", "Test2: v2"},
				},
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
			tc.Config.Client.Server = "test.com"

			err := tc.Config.ParseAndValidate()
			require.NoError(t, err)

			assert.Equal(t, tc.ExpectedHeader, tc.Config.Connection.Headers())
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
			ExpectedError: "Server address is required. See --help",
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
			ExpectedError: `Invalid server address: parse "http://test\n.com": net/url: invalid control character in URL`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.ServerURL, func(t *testing.T) {
			config := &Config{
				Client: ClientConfig{
					Server: tc.ServerURL,
				},
			}
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
