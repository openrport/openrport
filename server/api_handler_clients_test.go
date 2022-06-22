package chserver

import (
	"context"
	"fmt"
	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"
)

type mockClientGroupProvider struct {
	cgroups.ClientGroupProvider
}

func (mockClientGroupProvider) GetAll(ctx context.Context) ([]*cgroups.ClientGroup, error) {
	return nil, nil
}

func TestHandleGetClient(t *testing.T) {
	c1 := clients.New(t).ID("client-1").ClientAuthID(cl1.ID).Build()
	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			clientService: NewClientService(nil, nil, clients.NewClientRepository([]*clients.Client{c1}, &hour, testLog)),
			config: &Config{
				Server: ServerConfig{MaxRequestBytes: 1024 * 1024},
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
        "version":"0.1.12",
        "address":"88.198.189.161:50078",
        "timezone":"UTC-0",
        "tunnels":[
            {
                "protocol": "tcp",
                "lhost":"0.0.0.0",
                "lport":"2222",
                "rhost":"0.0.0.0",
                "rport":"22",
                "lport_random":false,
                "scheme":null,
                "acl":null,
                "host_header":"",
                "http_proxy":false,
		        "idle_timeout_minutes": 0,
		        "auto_close": 0,
                "id":"1"
            },
            {
                "protocol": "tcp",
                "lhost":"0.0.0.0",
                "lport":"4000",
                "rhost":"0.0.0.0",
                "rport":"80",
                "lport_random":false,
                "scheme":null,
                "acl":null,
                "host_header":"",
                "http_proxy":false,
		        "idle_timeout_minutes": 0,
		        "auto_close": 0,
                "id":"2"
            }
        ],
        "connection_state":"connected",
        "cpu_family":"Virtual CPU",
        "cpu_model":"Virtual CPU",
        "cpu_model_name":"",
        "cpu_vendor":"GenuineIntel",
        "disconnected_at":null,
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
	c1 := clients.New(t).ID("client-1").ClientAuthID(cl1.ID).Build()
	c2 := clients.New(t).ID("client-2").ClientAuthID(cl1.ID).DisconnectedDuration(5 * time.Minute).Build()
	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			clientService: NewClientService(nil, nil, clients.NewClientRepository([]*clients.Client{c1, c2}, &hour, testLog)),
			config: &Config{
				Server: ServerConfig{MaxRequestBytes: 1024 * 1024},
			},
			clientGroupProvider: mockClientGroupProvider{},
		},
		userService: users.NewAPIService(users.NewStaticProvider([]*users.User{curUser}), false),
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
