package chserver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/stretchr/testify/require"
)

// ED TODO: this INSIDE PLUS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
func TestValidateExtendedTunnelsPermissions(t *testing.T) {
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
					"host_header": "cray-1",
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
	// ED TODO: remove numbers in the test cases
	testCases := []struct {
		Name          string
		URL           string
		ExpectedError string
	}{
		{
			Name:          "missing local port",
			URL:           "/someurl?scheme=ssh&skip-idle-timeout=false&local=&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel without parameter local is forbidden. Allowed values: [20000 20001]",
		},
		{
			Name:          "wrong local port",
			URL:           "/someurl?scheme=ssh&skip-idle-timeout=false&local=8080&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel with parameter local=8080 is forbidden. Allowed values: [20000 20001]",
		},

		{
			Name:          "missing remote port",
			URL:           "/someurl?scheme=ssh&skip-idle-timeout=false&local=20000&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel without parameter remote is forbidden. Allowed values: [22 3389]",
		},
		{
			Name:          "wrong remote port",
			URL:           "/someurl?scheme=ssh&skip-idle-timeout=false&local=20000&remote=8080&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel with parameter remote=8080 is forbidden. Allowed values: [22 3389]",
		},

		{
			Name:          "missing scheme",
			URL:           "/someurl?scheme=&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel without parameter scheme is forbidden. Allowed values: [ssh rdp]",
		},
		{
			Name:          "wrong scheme",
			URL:           "/someurl?scheme=ftp&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel with parameter scheme=ftp is forbidden. Allowed values: [ssh rdp]",
		},

		{
			Name:          "missing acl",
			URL:           "/someurl?scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel without parameter acl is forbidden. Allowed values: [201.203.40.9]",
		},
		{
			Name:          "wrong acl",
			URL:           "/someurl?scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=202.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel with parameter acl=202.203.40.9 is forbidden. Allowed values: [201.203.40.9]",
		},

		{
			Name:          "idle-timeout-minutes is not set",
			URL:           "/someurl?scheme=ssh&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "4 Tunnel with idle-timeout-minutes=0 is forbidden. Allowed value must be greater than 5m",
		},
		{
			Name:          "idle-timeout-minutes is lower",
			URL:           "/someurl?scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=2&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "4 Tunnel with idle-timeout-minutes=2 is forbidden. Allowed value must be greater than 5m",
		},

		{
			Name:          "auto-close is not set / lower than expected",
			URL:           "/someurl?idle-timeout-minutes=5&scheme=ssh&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&auto-close=0&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "4 Tunnel with auto-close=0 is forbidden. Allowed value must be greater than 1m",
		},
		{
			Name:          "auto-close is higher than expected",
			URL:           "/someurl?idle-timeout-minutes=5&scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&auto-close=200m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "4 Tunnel with auto-close=200 is forbidden. Allowed value must be less than 60m",
		},

		{
			Name:          "missing protocol",
			URL:           "/someurl?protocol=&scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel without parameter protocol is forbidden. Allowed values: [tcp udp tcp-udp]",
		},
		{
			Name:          "wrong protocol",
			URL:           "/someurl?protocol=icmp&scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "3 Tunnel with parameter protocol=icmp is forbidden. Allowed values: [tcp udp tcp-udp]",
		},

		{
			Name:          "trying to set skip-idle-timeout",
			URL:           "/someurl?protocol=tcp&scheme=ssh&skip-idle-timeout=true&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "1 Tunnel with skip-idle-timeout=true is forbidden. You are not allowed to set skip-idle-timeout value",
		},

		{
			Name: "set http-proxy true",
			URL:  "/someurl?http-proxy=true&protocol=tcp&scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
		},
		{
			Name: "set http-proxy false",
			URL:  "/someurl?http-proxy=false&protocol=tcp&scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
		},

		{
			Name: "missing host_header",
			URL:  "/someurl?host_header=cray-1&protocol=tcp&scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
		},
		{
			Name:          "wrong host_header",
			URL:           "/someurl?host_header=aBc&protocol=tcp&scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&auth_allowed=true&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
			ExpectedError: "2 Tunnel with host_header=aBc is forbidden. Allowed values must match 'cray-1' regular expression",
		},

		{
			Name: "set auth_allowed true",
			URL:  "/someurl?auth_allowed=true&protocol=tcp&scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
		},
		{
			Name: "set auth_allowed false",
			URL:  "/someurl?auth_allowed=false&protocol=tcp&scheme=ssh&skip-idle-timeout=false&local=20000&remote=22&acl=201.203.40.9&host_header=cray-1&idle-timeout-minutes=20&auto-close=60m&protocol=tcp&protocol=udp&protocol=tcp-udp",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Logf("Test case: %s", tc.Name)
		inputURL, err := url.Parse(tc.URL)
		require.NoError(t, err)

		req := &http.Request{
			URL: inputURL,
		}
		err = validateExtendedTunnelPermission(req, restrictions)
		if tc.ExpectedError != "" {
			require.EqualError(t, err, tc.ExpectedError)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestValidateExtendedCommandPermission(t *testing.T) {
	var restrictions []users.StringInterfaceMap
	err := json.Unmarshal([]byte(`
			[
				{
					"allow": ["^sudo reboot$","^systemctl .* restart$"],
					"deny": ["apache2","ssh"],
					"is_sudo": false
				},
				{
					"deny": ["^systemctl .* restart$"],
					"is_sudo": false
				},
				{
					"deny": ["rm -rf /"],
					"is_sudo": true
				}
			]
	`), &restrictions)
	require.NoError(t, err)

	testCases := []struct {
		Name          string
		requestBody   string
		ExpectedError string
	}{
		{
			Name:          "command forbidden by DENY",
			requestBody:   `{"command": "ssh foo"}`,
			ExpectedError: "Command 'ssh foo' forbidden. Allowed values must not match DENY 'ssh' regular expressions: [apache2 ssh]",
		},
		{
			Name:          "command \"cmd\" forbidden by DENY",
			requestBody:   `{"cmd": "ssh foo"}`,
			ExpectedError: "Command 'ssh foo' forbidden. Allowed values must not match DENY 'ssh' regular expressions: [apache2 ssh]",
		},
		{
			Name:          "command forbidden by ALLOW",
			requestBody:   `{"command": "whoami"}`,
			ExpectedError: "Command 'whoami' forbidden. Allowed values must match one of ALLOW regular expressions: [^sudo reboot$ ^systemctl .* restart$]",
		},
		{
			Name:          "command forbidden by DENY (of another group)",
			requestBody:   `{"command": "systemctl .* restart"}`,
			ExpectedError: "Command 'systemctl .* restart' forbidden. Allowed values must not match DENY '^systemctl .* restart$' regular expressions: [^systemctl .* restart$]",
		},
		{
			Name:          "command forbidden by DENY (of another group)",
			requestBody:   `{"command": "rm -rf /"}`,
			ExpectedError: "Command 'rm -rf /' forbidden. Allowed values must match one of ALLOW regular expressions: [^sudo reboot$ ^systemctl .* restart$]",
		},
		{
			Name:        "command allowed",
			requestBody: `{"command": "sudo reboot"}`,
		},
		{
			Name:          "command forbidden by is_sudo flag",
			requestBody:   `{"cmd": "systemctl .* restart", "is_sudo": true}`,
			ExpectedError: "Command 'systemctl .* restart' forbidden. Allowed values must not use the global is_sudo switch",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Logf("Test case: %s", tc.Name)
		inputURL, err := url.Parse("/someurl")
		require.NoError(t, err)

		req := &http.Request{
			Method: http.MethodPost,
			URL:    inputURL,
			Body:   ioutil.NopCloser(strings.NewReader(tc.requestBody)),
		}

		err = validateExtendedCommandPermission(req, restrictions)
		if tc.ExpectedError != "" {
			require.EqualError(t, err, tc.ExpectedError)
		} else {
			require.NoError(t, err)
		}
	}
}
