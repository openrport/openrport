package chserver

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/stretchr/testify/require"
)

// ED TODO: this INSIDE PLUS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
func TestValidateExtendedTunnelPermissions(t *testing.T) {
	var restrictions []users.StringInterfaceMap
	err := json.Unmarshal([]byte(`
			[
				{
					"local": ["20000", "20001"],
					"remote": ["22", "3389"], 
					"scheme": ["ssh", "rdp"],
					"acl": ["201.203.40.9"],
					"idle-timeout-minutes": {
						"min": 5
					},
					"auto-close": {
						"max": "60m",
						"min": "1m"
					},
					"protocol": ["tcp", "udp", "tcp-udp"],
					"skip-idle-timeout": false,
					"http_proxy": true,
					"host_header": ":*",
					"auth_allowed": true
				},		
				{
					"local": ["20000", "20001", "20002"],
					"remote": ["22", "3389", "3390"],
					"scheme": ["ssh", "rdp", "ssh"],
					"acl": ["201.203.40.9"]
				}
			]
	`), &restrictions)
	require.NoError(t, err)

	testCases := []struct {
		Name          string
		URL           string
		ExpectedError string
	}{
		{
			Name:          "idle-timeout-minutes is not set",
			URL:           "/someurl?scheme=ssh&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=*&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "Tunnel with idle-timeout-minutes=0m is forbidden. Allowed value for user group must be greater than 5m",
		},
		{
			Name:          "idle-timeout-minutes is lower",
			URL:           "/someurl?scheme=ssh&skip-idle-timeout=false& local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=*&idle-timeout-minutes=2&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "Tunnel with idle-timeout-minutes=2 is forbidden. Allowed value for user group must be greater than 5m",
		},
	}

	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.Name)
		inputURL, err := url.Parse(tc.URL)
		require.NoError(t, err)

		req := &http.Request{
			URL: inputURL,
		}
		err = validateExtendedTunnelPermissions(req, restrictions)
		if tc.ExpectedError != "" {
			require.EqualError(t, err, tc.ExpectedError)
		} else {
			require.NoError(t, err)
		}
	}
}
