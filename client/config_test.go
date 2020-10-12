package chclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigParseAndValidate(t *testing.T) {
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
			err := tc.Config.ParseAndValidate()
			require.NoError(t, err)

			assert.Equal(t, tc.ExpectedHeader, tc.Config.Connection.Headers())
		})
	}
}
