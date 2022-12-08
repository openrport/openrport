package clients

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/server/cgroups"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"
)

func TestStartClient(t *testing.T) {
	connMock := test.NewConnMock()
	connMock.ReturnRemoteAddr = &net.TCPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 2345}

	testCases := []struct {
		Name              string
		ClientAuthID      string
		ClientID          string
		AuthMultiuseCreds bool
		ExpectedError     error
	}{
		{
			Name:          "existing client id same client auth",
			ClientAuthID:  "test-client-auth",
			ClientID:      "test-client",
			ExpectedError: errors.New("client is already connected:  [test-client]"),
		}, {
			Name:          "existing client id different client",
			ClientAuthID:  "test-client-auth-2",
			ClientID:      "test-client",
			ExpectedError: errors.New("client is already connected:  [test-client]"),
		}, {
			Name:          "existing client with different id for client auth",
			ClientAuthID:  "test-client-auth",
			ClientID:      "test-client-2",
			ExpectedError: errors.New("client auth ID is already in use: \"test-client-auth\""),
		}, {
			Name:          "no existing client",
			ClientAuthID:  "test-client-auth-2",
			ClientID:      "test-client-2",
			ExpectedError: nil,
		}, {
			Name:              "existing client id same client auth, auth multiuse",
			ClientAuthID:      "test-client-auth",
			ClientID:          "test-client",
			AuthMultiuseCreds: true,
			ExpectedError:     errors.New("client is already connected:  [test-client]"),
		}, {
			Name:              "existing client id different client auth, auth multiuse",
			ClientAuthID:      "test-client-auth-2",
			ClientID:          "test-client",
			AuthMultiuseCreds: true,
			ExpectedError:     errors.New("client is already connected:  [test-client]"),
		}, {
			Name:              "existing client with different id for client auth, auth multiuse",
			ClientAuthID:      "test-client-auth",
			ClientID:          "test-client-2",
			AuthMultiuseCreds: true,
			ExpectedError:     nil,
		}, {
			Name:              "no existing client, auth multiuse",
			ClientAuthID:      "test-client-auth-2",
			ClientID:          "test-client-2",
			AuthMultiuseCreds: true,
			ExpectedError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cs := &ClientServiceProvider{
				repo: NewClientRepository([]*Client{{
					ID:           "test-client",
					ClientAuthID: "test-client-auth",
				}}, nil, testLog),
				portDistributor: ports.NewPortDistributor(mapset.NewThreadUnsafeSet()),
			}
			_, err := cs.StartClient(
				context.Background(), tc.ClientAuthID, tc.ClientID, connMock, tc.AuthMultiuseCreds,
				&chshare.ConnectionRequest{}, testLog)
			assert.Equal(t, tc.ExpectedError, err)
		})
	}
}

func TestStartClientDisconnected(t *testing.T) {
	connMock := test.NewConnMock()
	connMock.ReturnRemoteAddr = &net.TCPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 2345}
	now := time.Now()
	cs := &ClientServiceProvider{
		repo: NewClientRepository([]*Client{{
			ID:                "disconnected-client",
			ClientAuthID:      "test-client-auth",
			DisconnectedAt:    &now,
			AllowedUserGroups: []string{"test-group"},
			UpdatesStatus:     &models.UpdatesStatus{UpdatesAvailable: 13},
		}}, nil, testLog),
		portDistributor: ports.NewPortDistributor(mapset.NewThreadUnsafeSet()),
	}
	client, err := cs.StartClient(
		context.Background(), "test-client-auth", "disconnected-client", connMock, false,
		&chshare.ConnectionRequest{Name: "new-connection", Version: "0.7.0"}, testLog)
	assert.NoError(t, err)

	assert.Nil(t, client.DisconnectedAt)
	assert.Equal(t, "disconnected-client", client.ID)
	assert.Equal(t, "new-connection", client.Name)
	assert.Equal(t, []string{"test-group"}, client.AllowedUserGroups)
	assert.Equal(t, 13, client.UpdatesStatus.UpdatesAvailable)
}

func TestDeleteOfflineClient(t *testing.T) {
	c1Active := New(t).Build()
	c2Active := New(t).Build()
	c3Offline := New(t).DisconnectedDuration(5 * time.Minute).Build()
	c4Offline := New(t).DisconnectedDuration(time.Minute).Build()

	testCases := []struct {
		name      string
		clientID  string
		wantError error
	}{
		{
			name:      "delete offline client",
			clientID:  c3Offline.ID,
			wantError: nil,
		},
		{
			name:     "delete active client",
			clientID: c1Active.ID,
			wantError: errors2.APIError{
				Message:    "Client is active, should be disconnected",
				HTTPStatus: http.StatusBadRequest,
			},
		},
		{
			name:     "delete unknown client",
			clientID: "unknown-id",
			wantError: errors2.APIError{
				Message:    fmt.Sprintf("Client with id=%q not found.", "unknown-id"),
				HTTPStatus: http.StatusNotFound,
			},
		},
		{
			name:     "empty client ID",
			clientID: "",
			wantError: errors2.APIError{
				Message:    "Client id is empty",
				HTTPStatus: http.StatusBadRequest,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			clientService := NewClientService(nil, nil, NewClientRepository([]*Client{c1Active, c2Active, c3Offline, c4Offline}, &hour, testLog), testLog)
			before, err := clientService.Count()
			require.NoError(t, err)
			require.Equal(t, 4, before)

			// when
			gotErr := clientService.DeleteOffline(tc.clientID)

			// then
			require.Equal(t, tc.wantError, gotErr)
			var wantAfter int
			if tc.wantError != nil {
				wantAfter = before
			} else {
				wantAfter = before - 1
			}
			gotAfter, err := clientService.Count()
			require.NoError(t, err)
			assert.Equal(t, wantAfter, gotAfter)
		})
	}
}

func TestCheckLocalPort(t *testing.T) {
	srv := ClientServiceProvider{
		portDistributor: ports.NewPortDistributorForTests(
			mapset.NewThreadUnsafeSetFromSlice([]interface{}{1, 2, 3, 4, 5}),
			mapset.NewThreadUnsafeSetFromSlice([]interface{}{2, 3, 4}),
			mapset.NewThreadUnsafeSetFromSlice([]interface{}{2, 3, 4, 5}),
		),
	}

	invalidPort := "24563a"
	_, invalidPortParseErr := strconv.Atoi(invalidPort)

	testCases := []struct {
		name      string
		port      string
		protocol  string
		wantError error
	}{
		{
			name:      "valid port",
			port:      "2",
			wantError: nil,
		},
		{
			name: "invalid port",
			port: invalidPort,
			wantError: errors2.APIError{
				Message:    "Invalid local port: 24563a.",
				Err:        invalidPortParseErr,
				HTTPStatus: http.StatusBadRequest,
			},
		},
		{
			name: "not allowed port",
			port: "6",
			wantError: errors2.APIError{
				Message:    "Local port 6 is not among allowed ports.",
				HTTPStatus: http.StatusBadRequest,
			},
		},
		{
			name: "busy port tcp",
			port: "5",
			wantError: errors2.APIError{
				Message:    "Local port 5 already in use.",
				HTTPStatus: http.StatusConflict,
			},
		},
		{
			name:      "udp port not busy",
			port:      "5",
			protocol:  models.ProtocolUDP,
			wantError: nil,
		},
		{
			name:     "tcp+udp port busy",
			port:     "5",
			protocol: models.ProtocolTCPUDP,
			wantError: errors2.APIError{
				Message:    "Local port 5 already in use.",
				HTTPStatus: http.StatusConflict,
			},
		},
		{
			name:      "tcp+udp port not busy",
			port:      "4",
			protocol:  models.ProtocolTCPUDP,
			wantError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.protocol == "" {
				tc.protocol = models.ProtocolTCP
			}
			// when
			gotErr := srv.checkLocalPort(tc.protocol, tc.port)

			// then
			require.Equal(t, tc.wantError, gotErr)
		})
	}
}

func TestCheckClientsAccess(t *testing.T) {
	c1 := New(t).Build()                                                             // no groups
	c2 := New(t).AllowedUserGroups([]string{users.Administrators}).Build()           // admin
	c3 := New(t).AllowedUserGroups([]string{users.Administrators, "group1"}).Build() // admin + group1
	c4 := New(t).AllowedUserGroups([]string{"group1"}).Build()                       // group1
	c5 := New(t).AllowedUserGroups([]string{"group1", "group2"}).Build()             // group1 + group2
	c6 := New(t).AllowedUserGroups([]string{"group3"}).Build()                       // group3
	c7 := New(t).Build()

	allClients := []*Client{c1, c2, c3, c4, c5, c6}
	clientGroups := []*cgroups.ClientGroup{
		{
			ID:                "1",
			AllowedUserGroups: []string{"group4"},
			Params: &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{cgroups.Param(c7.ID)},
			},
		},
	}

	testCases := []struct {
		name                      string
		clients                   []*Client
		user                      *users.User
		wantClientIDsWithNoAccess []string
	}{
		{
			name:                      "user with no groups has no access",
			clients:                   allClients,
			user:                      &users.User{Groups: nil},
			wantClientIDsWithNoAccess: []string{c1.ID, c2.ID, c3.ID, c4.ID, c5.ID, c6.ID},
		},
		{
			name:                      "admin user has access to all",
			clients:                   allClients,
			user:                      &users.User{Groups: []string{users.Administrators}},
			wantClientIDsWithNoAccess: nil,
		},
		{
			name:                      "non-admin user with access to all groups",
			clients:                   []*Client{c3, c4, c5, c6},
			user:                      &users.User{Groups: []string{"group1", "group2", "group3"}},
			wantClientIDsWithNoAccess: nil,
		},
		{
			name:                      "non-admin user with no access to clients with no groups and with admin group",
			clients:                   allClients,
			user:                      &users.User{Groups: []string{"group1", "group2", "group3"}},
			wantClientIDsWithNoAccess: []string{c1.ID, c2.ID},
		},
		{
			name:                      "non-admin user with access to one client",
			clients:                   allClients,
			user:                      &users.User{Groups: []string{"group3"}},
			wantClientIDsWithNoAccess: []string{c1.ID, c2.ID, c3.ID, c4.ID, c5.ID},
		},
		{
			name:                      "non-admin user with access to few clients",
			clients:                   allClients,
			user:                      &users.User{Groups: []string{"group1"}},
			wantClientIDsWithNoAccess: []string{c1.ID, c2.ID, c6.ID},
		},
		{
			name:                      "non-admin user that has unknown group",
			clients:                   allClients,
			user:                      &users.User{Groups: []string{"group4"}},
			wantClientIDsWithNoAccess: []string{c1.ID, c2.ID, c3.ID, c4.ID, c5.ID, c6.ID},
		},
		{
			name:                      "non-admin user given access via client groups",
			clients:                   []*Client{c7},
			user:                      &users.User{Groups: []string{"group4"}},
			wantClientIDsWithNoAccess: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			clientService := NewClientService(nil, nil, NewClientRepository(allClients, nil, testLog), testLog)

			// when
			gotErr := clientService.CheckClientsAccess(tc.clients, tc.user, clientGroups)

			// then
			if len(tc.wantClientIDsWithNoAccess) > 0 {
				wantErr := errors2.APIError{
					Message:    fmt.Sprintf("Access denied to client(s) with ID(s): %v", strings.Join(tc.wantClientIDsWithNoAccess, ", ")),
					HTTPStatus: http.StatusForbidden,
				}
				assert.Equal(t, wantErr, gotErr)
			} else {
				require.NoError(t, gotErr)
			}
		})
	}
}

func TestGetTunnelsToReestablish(t *testing.T) {
	var randomPorts = []string{"5001", "5002", "5003", "5004", "5005", "5006", "5007", "5008", "5009"}
	testCases := []struct {
		descr string // Test Case Description

		oldStr []string
		oldACL []string
		newStr []string
		newACL []string

		wantResStr []string
	}{
		{
			descr:      "both empty",
			oldStr:     nil,
			newStr:     nil,
			wantResStr: []string{},
		},
		{
			descr: "no new tunnels",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			newStr: []string{},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
				"0.0.0.0:3000:site.com:80",
				"::foobar.com:3000",
				"::127.0.0.1:3000",
			},
		},
		{
			descr:  "no old tunnels",
			oldStr: []string{},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			wantResStr: nil,
		},
		{
			descr: "same tunnels specified in 4 possible forms",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			wantResStr: nil,
		},
		{
			descr: "old tunnels include all new",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
				"192.168.0.1:3001:google.com:80",
				"3001:site.com:80",
				"foobar.com:3001",
				"3001",
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			wantResStr: []string{
				"192.168.0.1:3001:google.com:80",
				"0.0.0.0:3001:site.com:80",
				"::foobar.com:3001",
				"::127.0.0.1:3001",
			},
		},
		{
			descr: "old tunnels were with random ports, but new has the same random ports",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"3000",          //contains randomPorts[2]
				"foobar.com:22", //contains randomPorts[3]
				"3000",          //contains randomPorts[4]
			},
			newStr: []string{
				"0.0.0.0:" + randomPorts[1] + ":foobar.com:22",
				"0.0.0.0:" + randomPorts[2] + ":127.0.0.1:3000",
				"0.0.0.0:" + randomPorts[3] + ":foobar.com:22",
				"0.0.0.0:" + randomPorts[4] + ":127.0.0.1:3000",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
		},
		{
			descr: "old tunnels were with random ports, but new has 2 the same random ports and 2 random",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"3000",          //contains randomPorts[2]
				"foobar.com:22", //contains randomPorts[3]
				"3000",          //contains randomPorts[4]
			},
			newStr: []string{
				"0.0.0.0:" + randomPorts[1] + ":foobar.com:22",
				"0.0.0.0:" + randomPorts[2] + ":127.0.0.1:3000",
				"foobar.com:22",
				"3000",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
		},
		{
			descr: "old tunnels were with random ports, but new has the different random port",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"foobar.com:22", //contains randomPorts[2]
			},
			newStr: []string{
				"0.0.0.0:" + randomPorts[2] + ":foobar.com:22",
				"0.0.0.0:" + randomPorts[3] + ":foobar.com:22",
			},
			wantResStr: nil,
		},
		{
			descr: "old tunnels are with random port 1 and 2, new tunnels are with random port and a port that equals to random port 1",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"foobar.com:22", //contains randomPorts[2]
			},
			newStr: []string{
				"foobar.com:22",
				"0.0.0.0:" + randomPorts[1] + ":foobar.com:22",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
		},
		{
			descr: "old tunnels are with random port 1 and 2, new tunnels are with a port that equals to random port 1 and a random port",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"foobar.com:22", //contains randomPorts[2]
			},
			// different order to a previous test case
			newStr: []string{
				"0.0.0.0:" + randomPorts[1] + ":foobar.com:22",
				"foobar.com:22",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
		},
		{
			descr: "old tunnels include all new, multiple similar with random port",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"192.168.0.1:3000:google.com:8080",
				"3000:site.com:80",
				"foobar.com:3000", //contains randomPorts[4]
				"foobar.com:3000", //contains randomPorts[5]
				"foobar.com:3000", //contains randomPorts[6]
				"3000",            //contains randomPorts[7]
				"3000",            //contains randomPorts[8]
				"3000",            //contains randomPorts[9]
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"0.0.0.0:" + randomPorts[4] + ":foobar.com:3000",
				"foobar.com:3000",
				"foobar.com:3000",
				"3000",
				"3000",
				"0.0.0.0:" + randomPorts[7] + ":127.0.0.1:3000",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:8080",
			},
		},
		{
			descr: "new tunnels include all old",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
				"192.168.0.1:3001:google.com:80",
				"3001:site.com:80",
				"foobar.com:3001",
				"3001",
			},
			wantResStr: nil,
		},
		{
			descr: "new tunnel specified in form '<local-host>:<local-port>:<remote-host>:<remote-port>' is not among old",
			oldStr: []string{
				"192.168.0.2:3000:google.com:80",
				"192.168.0.1:3001:google.com:80",
				"192.168.0.1:3000:google.com.ua:80",
				"192.168.0.1:3000:google.com:8080",
				"3000:google.com:80",
				"google.com:80",
				"80",
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
			wantResStr: nil,
		},
		{
			descr: "new tunnel specified in form '<local-port>:<remote-host>:<remote-port>' is not among old",
			oldStr: []string{
				"192.168.0.1:3000:site.com:80",
				"3001:site.com:80",
				"3000:site-2.com:80",
				"3000:site.com:22",
				"site.com:80",
				"80",
			},
			newStr: []string{
				"3000:site.com:80",
			},
			wantResStr: nil,
		},
		{
			descr: "new tunnel specified in form '<remote-host>:<remote-port>' is not among old",
			oldStr: []string{
				"192.168.0.1:3000:foobar.com:3000",
				"0.0.0.0:3001:foobar.com:3000",
				"3000:foobar.com:3000",
				"foobar.com:3001",
				"foobar-2.com:3000",
				"3000",
			},
			newStr: []string{
				"foobar.com:3000",
			},
			wantResStr: nil,
		},
		{
			descr: "new tunnel specified in form '<remote-port>' is not among old",
			oldStr: []string{
				"192.168.0.1:3000:foobar.com:3000",
				"0.0.0.0:3000:127.0.0.1:3000",
				"3000:127.0.0.1:3000",
				"3000:foobar.com:3000",
				"foobar.com:3000",
				"3001",
			},
			newStr: []string{
				"3000",
			},
			wantResStr: nil,
		},
		{
			descr: "same old and new tunnel but different ACLs",
			oldStr: []string{
				"5432:127.0.0.1:22",
			},
			oldACL: []string{
				"95.67.52.213",
			},
			newStr: []string{
				"5432:127.0.0.1:22",
			},
			newACL: []string{
				"95.67.52.214",
			},
			wantResStr: nil,
		},
		{
			descr: "same old and new tunnel without local but different ACLs",
			oldStr: []string{
				"22",
			},
			oldACL: []string{
				"95.67.52.213",
			},
			newStr: []string{
				"22",
			},
			newACL: []string{
				"95.67.52.214",
			},
			wantResStr: nil,
		},
		{
			descr: "old tunnels have 2 similar tunnels but different ACLs, new tunnels contains one of them",
			oldStr: []string{
				"2222:127.0.0.1:22",
				"3333:127.0.0.1:22",
			},
			oldACL: []string{
				"95.67.52.213",
				"95.67.52.214",
			},
			newStr: []string{
				"2222:127.0.0.1:22",
			},
			newACL: []string{
				"95.67.52.213",
			},
			wantResStr: []string{
				"0.0.0.0:3333:127.0.0.1:22(acl:95.67.52.214)",
			},
		},
		{
			descr: "old and new tunnels have 2 same tunnels without local but different ACLs",
			oldStr: []string{
				"22",
				"22",
			},
			oldACL: []string{
				"95.67.52.213",
				"95.67.52.214",
			},
			newStr: []string{
				"22",
				"22",
			},
			newACL: []string{
				"95.67.52.213",
				"95.67.52.214",
			},
			wantResStr: nil,
		},
		{
			descr: "old tunnels have 3 same tunnels without local but different ACLs, new tunnels have 2 of them",
			oldStr: []string{
				"22",
				"22",
				"22",
			},
			oldACL: []string{
				"95.67.52.213",
				"95.67.52.214",
				"95.67.52.215",
			},
			newStr: []string{
				"22",
				"22",
			},
			newACL: []string{
				"95.67.52.213",
				"95.67.52.214",
			},
			wantResStr: []string{
				"::127.0.0.1:22(acl:95.67.52.215)",
			},
		},
	}
	for _, tc := range testCases {
		msg := fmt.Sprintf("test case: %q", tc.descr)

		// given
		var old, new []*models.Remote
		for i, v := range tc.oldStr {
			r, err := models.NewRemote(v)
			require.NoErrorf(t, err, msg)
			// mimic real behavior
			if !r.IsLocalSpecified() {
				r.LocalHost = "0.0.0.0"
				r.LocalPort = randomPorts[i]
				r.LocalPortRandom = true
			}
			if tc.oldACL != nil && tc.oldACL[i] != "" {
				r.ACL = &tc.oldACL[i]
			}
			old = append(old, r)
		}
		for i, v := range tc.newStr {
			r, err := models.NewRemote(v)
			require.NoErrorf(t, err, msg)
			if tc.newACL != nil && tc.newACL[i] != "" {
				r.ACL = &tc.newACL[i]
			}
			new = append(new, r)
		}

		// when
		gotRes := GetTunnelsToReestablish(old, new)

		var gotResStr []string
		for _, r := range gotRes {
			gotResStr = append(gotResStr, r.String())
		}

		// then
		assert.ElementsMatch(t, tc.wantResStr, gotResStr, msg)
	}
}
