package chserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/caddy"
	"github.com/realvnc-labs/rport/server/cgroups"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/server/clients"
	"github.com/realvnc-labs/rport/server/clients/clienttunnel"
	"github.com/realvnc-labs/rport/share/models"
	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/test"
)

type mockClientGroupProvider struct {
	cgroups.ClientGroupProvider
}

func (mockClientGroupProvider) GetAll(ctx context.Context) ([]*cgroups.ClientGroup, error) {
	return nil, nil
}

func TestHandleGetClient(t *testing.T) {
	c1 := clients.New(t).ID("client-1").ClientAuthID(cl1.ID).Logger(testLog).Build()
	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			clientService: clients.NewClientService(nil, nil, clients.NewClientRepository([]*clients.Client{c1}, &hour, testLog), testLog, nil),
			config: &chconfig.Config{
				API: chconfig.APIConfig{
					MaxRequestBytes: 1024 * 1024,
				},
			},
			clientGroupProvider: mockClientGroupProvider{},
		},
	}

	al.initRouter()

	testCases := []struct {
		Name           string
		URL            string
		ExpectedStatus int
	}{
		{
			Name:           "Ok",
			URL:            "/api/v1/clients/client-1",
			ExpectedStatus: http.StatusOK,
		}, {
			Name:           "Not found",
			URL:            "/api/v1/clients/client-2",
			ExpectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tc.URL, nil)
			al.router.ServeHTTP(w, req)

			expectedJSON := `{
    "data":{
        "id":"client-1",
        "mem_total":100000,
        "name":"Random Rport Client",
        "num_cpus":2,
        "os":"Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
        "os_arch":"amd64",
        "os_family":"alpine",
        "os_full_name":"Debian 18.0",
        "os_kernel":"linux",
        "os_version":"18.0",
        "os_virtualization_role":"guest",
        "os_virtualization_system":"LVM",
        "hostname":"alpine-3-10-tk-01",
        "ipv4":[
            "192.168.122.111"
        ],
        "ipv6":[
            "fe80::b84f:aff:fe59:a0b1"
        ],
        "tags":[
            "Linux",
            "Datacenter 1"
        ],
        "labels": {"country": "Germany", "city": "Cologne", "datacenter": "NetCologne GmbH" },
        "version":"0.1.12",
        "address":"88.198.189.161:50078",
        "timezone":"UTC-0",
        "tunnels":[
            {
                "name": "",
                "owner": "",
                "protocol": "tcp",
                "lhost":"0.0.0.0",
                "lport":"2222",
                "rhost":"0.0.0.0",
                "rport":"22",
                "lport_random":false,
                "scheme":null,
                "acl":null,
                "host_header":"",
                "auth_user":"",
                "auth_password":"",
                "http_proxy":false,
                "idle_timeout_minutes": 0,
                "auto_close": 0,
                "created_at":"0001-01-01T00:00:00Z",
                "id":"1",
                "tunnel_url":""
            },
            {
                "name": "",
                "owner": "",
                "protocol": "tcp",
                "lhost":"0.0.0.0",
                "lport":"4000",
                "rhost":"0.0.0.0",
                "rport":"80",
                "lport_random":false,
                "scheme":null,
                "acl":null,
                "host_header":"",
                "auth_user":"",
                "auth_password":"",
                "http_proxy":false,
                "idle_timeout_minutes": 0,
                "auto_close": 0,
                "created_at":"0001-01-01T00:00:00Z",
                "id":"2",
                "tunnel_url":""
            }
        ],
        "connection_state":"connected",
        "cpu_family":"Virtual CPU",
        "cpu_model":"Virtual CPU",
        "cpu_model_name":"",
        "cpu_vendor":"GenuineIntel",
        "disconnected_at":null,
        "last_heartbeat_at":null,
        "client_auth_id":"user1",
        "allowed_user_groups":null,
        "updates_status":null,
        "client_configuration":null,
        "groups": []
    }
}`
			assert.Equal(t, tc.ExpectedStatus, w.Code)
			if tc.ExpectedStatus == http.StatusOK {
				assert.JSONEq(t, expectedJSON, w.Body.String())
			}
		})
	}
}

func TestHandleGetClients(t *testing.T) {
	curUser := &users.User{
		Username: "admin",
		Groups:   []string{users.Administrators},
	}

	c1 := clients.New(t).ID("client-1").ClientAuthID(cl1.ID).Logger(testLog).Build()
	c2 := clients.New(t).ID("client-2").ClientAuthID(cl1.ID).DisconnectedDuration(5 * time.Minute).Logger(testLog).Build()

	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			clientService: clients.NewClientService(nil, nil, clients.NewClientRepository([]*clients.Client{c1, c2}, &hour, testLog), testLog, nil),
			config: &chconfig.Config{
				API: chconfig.APIConfig{
					MaxRequestBytes: 1024 * 1024,
				},
			},
			clientGroupProvider: mockClientGroupProvider{},
		},
		userService: users.NewAPIService(users.NewStaticProvider([]*users.User{curUser}), false, 0, -1),
	}
	al.initRouter()

	testCases := []struct {
		Name         string
		Offset       int
		Limit        int
		ExpectedJSON string
	}{
		{
			Name: "regular",
			ExpectedJSON: `{
   "data":[
      {
         "id":"client-1",
         "name":"Random Rport Client",
         "hostname":"alpine-3-10-tk-01"
      },
      {
         "id":"client-2",
         "name":"Random Rport Client",
         "hostname":"alpine-3-10-tk-01"
      }
   ],
   "meta": {"count": 2}
}`,
		},
		{
			Name:  "limit",
			Limit: 1,
			ExpectedJSON: `{
   "data":[
      {
         "id":"client-1",
         "name":"Random Rport Client",
         "hostname":"alpine-3-10-tk-01"
      }
   ],
   "meta": {"count": 2}
}`,
		},
		{
			Name:   "limit+offset",
			Limit:  1,
			Offset: 1,
			ExpectedJSON: `{
   "data":[
      {
         "id":"client-2",
         "name":"Random Rport Client",
         "hostname":"alpine-3-10-tk-01"
      }
   ],
   "meta": {"count": 2}
}`,
		},
		{
			Name:   "large offset and limit",
			Offset: 100,
			Limit:  100,
			ExpectedJSON: `{
   "data":[],
   "meta": {"count": 2}
}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			v := url.Values{}
			if tc.Limit > 0 {
				v.Set("page[limit]", strconv.Itoa(tc.Limit))
			}
			if tc.Offset > 0 {
				v.Set("page[offset]", strconv.Itoa(tc.Offset))
			}
			req := httptest.NewRequest("GET", "/api/v1/clients?"+v.Encode(), nil)
			ctx := api.WithUser(context.Background(), curUser.Username)
			req = req.WithContext(ctx)
			al.router.ServeHTTP(w, req)

			assert.Equal(t, 200, w.Code)
			assert.JSONEq(t, tc.ExpectedJSON, w.Body.String())
		})
	}
}

func TestGetCorrespondingSortFuncPositive(t *testing.T) {
	testCases := []struct {
		sortStr string

		wantFunc func(a []*clients.CalculatedClient, desc bool)
		wantDesc bool
	}{
		{
			sortStr:  "",
			wantFunc: clients.SortByID,
			wantDesc: false,
		},
		{
			sortStr:  "id",
			wantFunc: clients.SortByID,
			wantDesc: false,
		},
		{
			sortStr:  "-id",
			wantFunc: clients.SortByID,
			wantDesc: true,
		},
		{
			sortStr:  "name",
			wantFunc: clients.SortByName,
			wantDesc: false,
		},
		{
			sortStr:  "-name",
			wantFunc: clients.SortByName,
			wantDesc: true,
		},
		{
			sortStr:  "hostname",
			wantFunc: clients.SortByHostname,
			wantDesc: false,
		},
		{
			sortStr:  "-hostname",
			wantFunc: clients.SortByHostname,
			wantDesc: true,
		},
		{
			sortStr:  "os",
			wantFunc: clients.SortByOS,
			wantDesc: false,
		},
		{
			sortStr:  "-os",
			wantFunc: clients.SortByOS,
			wantDesc: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.sortStr, func(t *testing.T) {
			t.Parallel()

			// when
			sortOptions := query.ParseSortOptions(map[string][]string{"sort": {tc.sortStr}})
			gotFunc, gotDesc, gotErr := getCorrespondingSortFunc(sortOptions)

			// then
			// workaround to compare func vars, see https://github.com/stretchr/testify/issues/182
			wantFuncName := runtime.FuncForPC(reflect.ValueOf(tc.wantFunc).Pointer()).Name()
			gotFuncName := runtime.FuncForPC(reflect.ValueOf(gotFunc).Pointer()).Name()
			msg := fmt.Sprintf("getCorrespondingSortFunc(%q) = (%s, %v, %v), expected: (%s, %v, %v)", tc.sortStr, gotFuncName, gotDesc, gotErr, wantFuncName, tc.wantDesc, nil)

			assert.NoErrorf(t, gotErr, msg)
			assert.Equalf(t, wantFuncName, gotFuncName, msg)
			assert.Equalf(t, tc.wantDesc, gotDesc, msg)
		})
	}
}

func TestGetCorrespondingSortFuncError(t *testing.T) {
	// when
	sortOptions := query.ParseSortOptions(map[string][]string{"sort": {"id", "-name"}})
	_, _, gotErr := getCorrespondingSortFunc(sortOptions)

	// then
	require.Error(t, gotErr)
	assert.Equal(t, gotErr.Error(), "Only one sort field is supported for clients.")
}

type SimpleMockClientService struct {
	ExpectedIDs   []string
	ActiveClients []*clients.Client

	*clients.ClientServiceProvider
}

func (mcs *SimpleMockClientService) GetActiveByID(id string) (*clients.Client, error) {
	// for this test, just return the first client
	return mcs.ActiveClients[0], nil
}

func (mcs *SimpleMockClientService) StartClientTunnels(client *clients.Client, remotes []*models.Remote) ([]*clienttunnel.Tunnel, error) {
	tunnels := make([]*clienttunnel.Tunnel, 0, 32)
	for i, remote := range remotes {
		tunnels = append(tunnels, makeTunnelResponse(mcs.ExpectedIDs[i], remote))
	}
	return tunnels, nil
}

func makeTunnelResponse(id string, remote *models.Remote) (response *clienttunnel.Tunnel) {
	response = &clienttunnel.Tunnel{
		ID:     id,
		Remote: *remote,
	}
	return response
}

func TestHandlePutTunnelWithName(t *testing.T) {
	connMock := test.NewConnMock()
	connMock.ReturnOk = true
	connMock.ReturnResponsePayload = []byte("{ \"IsAllowed\": true }")
	user := &users.User{
		Username: "test-user",
	}
	mockUsersService := &MockUsersService{
		UserService: users.NewAPIService(users.NewStaticProvider([]*users.User{user}), false, 0, -1),
	}

	testCases := []struct {
		Name          string
		URL           string
		ExpectedJSON  string
		ExpectedError string
	}{
		{
			Name: "With Name",
			URL:  "/api/v1/clients/client-1/tunnels?scheme=ssh&acl=127.0.0.1&local=0.0.0.0%3A3390&remote=0.0.0.0%3A22&name=TUNNELNAME&check_port=0",
			ExpectedJSON: `{
			"data": {
				"id": "10",
				"name": "TUNNELNAME",
				"owner": "test-user",
				"protocol": "tcp",
				"lhost": "0.0.0.0",
				"lport": "3390",
				"rhost": "0.0.0.0",
				"rport": "22",
				"lport_random": false,
				"scheme": "ssh",
				"acl": "127.0.0.1",
				"idle_timeout_minutes": 5,
				"auto_close": 0,
				"http_proxy": false,
				"host_header": "",
				"auth_user":"",
				"auth_password":"",
				"created_at": "0001-01-01T00:00:00Z",
				"tunnel_url": ""
			}
		}`,
		},
		{
			Name: "Without Name",
			URL:  "/api/v1/clients/client-1/tunnels?scheme=ssh&acl=127.0.0.1&local=0.0.0.0%3A3390&remote=0.0.0.0%3A22&check_port=0",
			ExpectedJSON: `{
			"data": {
				"id": "10",
				"name": "",
				"owner": "test-user",
				"protocol": "tcp",
				"lhost": "0.0.0.0",
				"lport": "3390",
				"rhost": "0.0.0.0",
				"rport": "22",
				"lport_random": false,
				"scheme": "ssh",
				"acl": "127.0.0.1",
				"idle_timeout_minutes": 5,
				"auto_close": 0,
				"http_proxy": false,
				"host_header": "",
				"auth_user":"",
				"auth_password":"",
				"created_at": "0001-01-01T00:00:00Z",
				"tunnel_url": ""
			}
		}`,
		},
		{
			Name: "Without Scheme",
			URL:  "/api/v1/clients/client-1/tunnels?acl=127.0.0.1&local=0.0.0.0%3A3390&remote=3000%3A22&check_port=0",
			ExpectedJSON: `{
			"data": {
				"id": "10",
				"name": "",
				"owner": "test-user",
				"protocol": "tcp",
				"lhost": "0.0.0.0",
				"lport": "3390",
				"rhost": "0.0.0.0",
				"rport": "22",
				"lport_random": false,
				"scheme": null,
				"acl": "127.0.0.1",
				"idle_timeout_minutes": 5,
				"auto_close": 0,
				"http_proxy": false,
				"host_header": "",
				"auth_user":"",
				"auth_password":"",
				"created_at": "0001-01-01T00:00:00Z",
				"tunnel_url": ""
			}
		}`,
		},
		{
			Name: "Without Name With User and Password",
			URL:  "/api/v1/clients/client-1/tunnels?scheme=http&acl=127.0.0.1&local=0.0.0.0%3A3390&remote=0.0.0.0%3A22&check_port=0&auth_user=admin&auth_password=foo&http_proxy=1",
			ExpectedJSON: `{
			"data": {
				"id": "10",
				"name": "",
				"owner": "test-user",
				"protocol": "tcp",
				"lhost": "0.0.0.0",
				"lport": "3390",
				"rhost": "0.0.0.0",
				"rport": "22",
				"lport_random": false,
				"scheme": "http",
				"acl": "127.0.0.1",
				"idle_timeout_minutes": 5,
				"auto_close": 0,
				"http_proxy": true,
				"host_header": "",
				"auth_user":"admin",
				"auth_password":"foo",
				"created_at": "0001-01-01T00:00:00Z",
				"tunnel_url": ""
			}
		}`,
		},
		{
			Name:          "Auth with error",
			URL:           "/api/v1/clients/client-1/tunnels?scheme=http&acl=127.0.0.1&local=0.0.0.0%3A3390&remote=0.0.0.0%3A22&check_port=0&auth_user=admin&http_proxy=1",
			ExpectedError: "auth_user requires auth_password",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			c1 := clients.New(t).ID("client-1").ClientAuthID(cl1.ID).Logger(testLog).Build()
			c1.SetConnection(connMock)
			c1.Logger = testLog

			mockClientService := &SimpleMockClientService{
				ExpectedIDs: []string{"10"},
				ActiveClients: []*clients.Client{
					c1,
				},
			}

			al := APIListener{
				insecureForTests: true,
				Server: &Server{
					clientService: mockClientService,
					config: &chconfig.Config{
						Server: chconfig.ServerConfig{
							InternalTunnelProxyConfig: clienttunnel.InternalTunnelProxyConfig{
								Enabled: true,
							},
						},
						API: chconfig.APIConfig{
							MaxRequestBytes: 1024 * 1024,
						},
					},
					clientGroupProvider: mockClientGroupProvider{},
				},
				userService: mockUsersService,
				Logger:      testLog,
			}
			al.initRouter()

			w := httptest.NewRecorder()
			req := httptest.NewRequest("PUT", tc.URL, nil)
			ctx := api.WithUser(req.Context(), user.Username)
			req = req.WithContext(ctx)

			al.router.ServeHTTP(w, req)
			if tc.ExpectedError == "" {
				assert.Equal(t, http.StatusOK, w.Code, fmt.Sprintf("Response Body: %s", w.Body))
				assert.JSONEq(t, tc.ExpectedJSON, w.Body.String())
			} else {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), tc.ExpectedError)
			}

		})
	}
}

func TestHandlePutTunnelUsingCaddyProxies(t *testing.T) {
	connMock := test.NewConnMock()
	connMock.ReturnOk = true
	connMock.ReturnResponsePayload = []byte("{ \"IsAllowed\": true }")
	user := &users.User{
		Username: "test-user",
	}
	mockUsersService := &MockUsersService{
		UserService: users.NewAPIService(users.NewStaticProvider([]*users.User{user}), false, 0, -1),
	}

	testCases := []struct {
		Name               string
		URL                string
		CaddyHostAddress   string
		ExpectedJSON       string
		ExpectedError      string
		DisableCaddyConfig bool
	}{
		{
			Name:             "With no subdomain",
			URL:              "/api/v1/clients/client-1/tunnels?scheme=ssh&acl=127.0.0.1&local=0.0.0.0%3A3390&remote=0.0.0.0%3A22&name=TUNNELNAME&check_port=0",
			CaddyHostAddress: "0.0.0.0:443",
			ExpectedJSON: `{
			"data": {
				"id": "10",
				"name": "TUNNELNAME",
				"owner": "test-user",
				"protocol": "tcp",
				"lhost": "0.0.0.0",
				"lport": "3390",
				"rhost": "0.0.0.0",
				"rport": "22",
				"lport_random": false,
				"scheme": "ssh",
				"acl": "127.0.0.1",
				"idle_timeout_minutes": 5,
				"auto_close": 0,
				"http_proxy": false,
				"host_header": "",
				"auth_user":"",
				"auth_password":"",
				"created_at": "0001-01-01T00:00:00Z",
				"tunnel_url": ""
			}
		}`,
		},
		{
			Name:             "With Subdomain",
			URL:              "/api/v1/clients/client-1/tunnels?scheme=http&acl=127.0.0.1&local=0.0.0.0%3A3390&remote=0.0.0.0%3A22&check_port=0&http_proxy=true",
			CaddyHostAddress: "0.0.0.0:443",
			ExpectedJSON: `{
			"data": {
				"id": "10",
				"name": "",
				"owner": "test-user",
				"protocol": "tcp",
				"lhost": "0.0.0.0",
				"lport": "3390",
				"rhost": "0.0.0.0",
				"rport": "22",
				"lport_random": false,
				"scheme": "http",
				"tunnel_url": "https://12345678.tunnels.rport.test:443",
				"acl": "127.0.0.1",
				"idle_timeout_minutes": 5,
				"auto_close": 0,
				"http_proxy": true,
				"host_header": "",
				"auth_user":"",
				"auth_password":"",
				"created_at": "0001-01-01T00:00:00Z"
			}
		}`,
		},
		{
			Name:             "With Subdomain and non-443 port",
			URL:              "/api/v1/clients/client-1/tunnels?scheme=http&acl=127.0.0.1&local=0.0.0.0%3A3390&remote=0.0.0.0%3A22&check_port=0&http_proxy=true",
			CaddyHostAddress: "0.0.0.0:8443",
			ExpectedJSON: `{
			"data": {
				"id": "10",
				"name": "",
				"owner": "test-user",
				"protocol": "tcp",
				"lhost": "0.0.0.0",
				"lport": "3390",
				"rhost": "0.0.0.0",
				"rport": "22",
				"lport_random": false,
				"scheme": "http",
				"tunnel_url": "https://12345678.tunnels.rport.test:8443",
				"acl": "127.0.0.1",
				"idle_timeout_minutes": 5,
				"auto_close": 0,
				"http_proxy": true,
				"host_header": "",
				"auth_user":"",
				"auth_password":"",
				"created_at": "0001-01-01T00:00:00Z"
			}
		}`,
		},
		{
			Name:             "With Subdomain, Without Scheme",
			URL:              "/api/v1/clients/client-1/tunnels?&acl=127.0.0.1&local=0.0.0.0%3A3390&remote=22&check_port=0&http_proxy=true",
			CaddyHostAddress: "0.0.0.0:443",
			ExpectedJSON: `{
			"data": {
				"id": "10",
				"name": "",
				"owner": "test-user",
				"protocol": "tcp",
				"lhost": "0.0.0.0",
				"lport": "3390",
				"rhost": "0.0.0.0",
				"rport": "22",
				"lport_random": false,
				"scheme": null,
				"tunnel_url": "https://12345678.tunnels.rport.test:443",
				"acl": "127.0.0.1",
				"idle_timeout_minutes": 5,
				"auto_close": 0,
				"http_proxy": true,
				"host_header": "",
				"auth_user":"",
				"auth_password":"",
				"created_at": "0001-01-01T00:00:00Z"
			}
		}`,
		},
		{
			Name:             "With HTTP proxy, Caddy Not Configured",
			CaddyHostAddress: "0.0.0.0:443",
			URL:              "/api/v1/clients/client-1/tunnels?scheme=http&acl=127.0.0.1&local=0.0.0.0%3A3390&remote=0.0.0.0%3A22&check_port=0&http_proxy=true",
			ExpectedJSON: `{
				"data": {
					"id": "10",
					"name": "",
					"owner": "test-user",
					"protocol": "tcp",
					"lhost": "0.0.0.0",
					"lport": "3390",
					"rhost": "0.0.0.0",
					"rport": "22",
					"lport_random": false,
					"scheme": "http",
					"tunnel_url": "",
					"acl": "127.0.0.1",
					"idle_timeout_minutes": 5,
					"auto_close": 0,
					"http_proxy": true,
					"host_header": "",
					"auth_user":"",
					"auth_password":"",
					"created_at": "0001-01-01T00:00:00Z"
				}
			}`,
			DisableCaddyConfig: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			c1 := clients.New(t).ID("client-1").ClientAuthID(cl1.ID).Logger(testLog).Build()
			c1.SetConnection(connMock)
			c1.Logger = testLog

			mockClientService := &SimpleMockClientService{
				ExpectedIDs: []string{"10"},
				ActiveClients: []*clients.Client{
					c1,
				},
			}

			al := APIListener{
				insecureForTests: true,
				Server: &Server{
					clientService: mockClientService,
					config: &chconfig.Config{
						Server: chconfig.ServerConfig{
							InternalTunnelProxyConfig: clienttunnel.InternalTunnelProxyConfig{
								Enabled: true,
							},
						},
						API: chconfig.APIConfig{
							MaxRequestBytes: 1024 * 1024,
						},
						Caddy: caddy.Config{
							ExecPath:         "/usr/bin/caddy",
							DataDir:          "/tmp",
							BaseConfFilename: "caddy-base.conf",
							HostAddress:      tc.CaddyHostAddress,
							BaseDomain:       "tunnels.rport.test",
							CertFile:         "../../testdata/certs/tunnels.rport.test.crt",
							KeyFile:          "../../testdata/certs/tunnels.rport.test.key",
							Enabled:          true,
						},
					},
					clientGroupProvider: mockClientGroupProvider{},
				},
				userService: mockUsersService,
				Logger:      testLog,
			}
			al.initRouter()

			originalGenerator := al.Server.config.Caddy.SubDomainGenerator
			al.Server.config.Caddy.SubDomainGenerator = &MockSubdomainGenerator{}

			if tc.DisableCaddyConfig {
				al.Server.config.Caddy.Enabled = false
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest("PUT", tc.URL, nil)
			ctx := api.WithUser(req.Context(), user.Username)
			req = req.WithContext(ctx)

			al.router.ServeHTTP(w, req)

			if tc.ExpectedError == "" {
				assert.Equal(t, http.StatusOK, w.Code, fmt.Sprintf("Response Body: %s", w.Body))
				assert.JSONEq(t, tc.ExpectedJSON, w.Body.String())
			} else {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), tc.ExpectedError)
			}

			// be good and restore the original generator
			al.Server.config.Caddy.SubDomainGenerator = originalGenerator
		})
	}
}

type MockSubdomainGenerator struct{}

func NewMockConfigWithSubdomainGenerator() (m *MockSubdomainGenerator) {
	return &MockSubdomainGenerator{}
}

func (m *MockSubdomainGenerator) GetRandomSubdomain() (subdomain string, err error) {
	return "12345678", nil
}

func TestHandlePutClientTunnelACL(t *testing.T) {
	mockTunnelProtocol := &MockTunnelProtocol{}
	c1 := clients.New(t).ID("client-1").Build()
	c1.Tunnels[0].TunnelProtocol = mockTunnelProtocol
	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			clientService: clients.NewClientService(nil, nil, clients.NewClientRepository([]*clients.Client{c1}, &hour, testLog), testLog, nil),
			config: &chconfig.Config{
				API: chconfig.APIConfig{
					MaxRequestBytes: 1024 * 1024,
				},
			},
		},
	}

	al.initRouter()

	acl, err := clienttunnel.ParseTunnelACL("127.0.0.0/24")
	require.NoError(t, err)

	testCases := []struct {
		Name           string
		URL            string
		Body           string
		ExpectedStatus int
		ExpectedACL    *clienttunnel.TunnelACL
	}{
		{
			Name:           "null acl",
			URL:            "/api/v1/clients/client-1/tunnels/1/acl",
			Body:           `{"acl": null}`,
			ExpectedStatus: http.StatusNoContent,
			ExpectedACL:    nil,
		}, {
			Name:           "valid acl",
			URL:            "/api/v1/clients/client-1/tunnels/1/acl",
			Body:           `{"acl": "127.0.0.0/24"}`,
			ExpectedStatus: http.StatusNoContent,
			ExpectedACL:    acl,
		}, {
			Name:           "invalid acl",
			URL:            "/api/v1/clients/client-1/tunnels/1/acl",
			Body:           `{"acl": "invalid"}`,
			ExpectedStatus: http.StatusBadRequest,
		}, {
			Name:           "unknown tunnel",
			URL:            "/api/v1/clients/client-1/tunnels/unknown/acl",
			Body:           `{"acl": null}`,
			ExpectedStatus: http.StatusNotFound,
		}, {
			Name:           "unknown client",
			URL:            "/api/v1/clients/unknown/tunnels/1/acl",
			Body:           `{"acl": null}`,
			ExpectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("PUT", tc.URL, strings.NewReader(tc.Body))
			al.router.ServeHTTP(w, req)

			assert.Equal(t, tc.ExpectedStatus, w.Code)
			if tc.ExpectedStatus == http.StatusNoContent {
				assert.Equal(t, tc.ExpectedACL, mockTunnelProtocol.ACL)
			}
		})
	}
}

type MockTunnelProtocol struct {
	clienttunnel.TunnelProtocol
	ACL *clienttunnel.TunnelACL
}

func (m *MockTunnelProtocol) SetACL(acl *clienttunnel.TunnelACL) {
	m.ACL = acl
}
