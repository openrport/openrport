package chserver

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

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/clients"
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
			ExpectedError: errors.New("client is already connected: test-client"),
		}, {
			Name:          "existing client id different client",
			ClientAuthID:  "test-client-auth-2",
			ClientID:      "test-client",
			ExpectedError: errors.New("client is already connected: test-client"),
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
			ExpectedError:     errors.New("client is already connected: test-client"),
		}, {
			Name:              "existing client id different client auth, auth multiuse",
			ClientAuthID:      "test-client-auth-2",
			ClientID:          "test-client",
			AuthMultiuseCreds: true,
			ExpectedError:     errors.New("client is already connected: test-client"),
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
			cs := &ClientService{
				repo: clients.NewClientRepository([]*clients.Client{{
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
	cs := &ClientService{
		repo: clients.NewClientRepository([]*clients.Client{{
			ID:                "disconnected-client",
			ClientAuthID:      "test-client-auth",
			DisconnectedAt:    &now,
			AllowedUserGroups: []string{"test-group"},
			UpdatesStatus:     &models.UpdatesStatus{UpdatesAvailable: 13},
			Version:           "0.7.0",
		}}, nil, testLog),
		portDistributor: ports.NewPortDistributor(mapset.NewThreadUnsafeSet()),
	}
	client, err := cs.StartClient(
		context.Background(), "test-client-auth", "disconnected-client", connMock, false,
		&chshare.ConnectionRequest{Name: "new-connection"}, testLog)
	assert.NoError(t, err)

	assert.Nil(t, client.DisconnectedAt)
	assert.Equal(t, "disconnected-client", client.ID)
	assert.Equal(t, "new-connection", client.Name)
	assert.Equal(t, []string{"test-group"}, client.AllowedUserGroups)
	assert.Equal(t, 13, client.UpdatesStatus.UpdatesAvailable)
}

func TestDeleteOfflineClient(t *testing.T) {
	c1Active := clients.New(t).Build()
	c2Active := clients.New(t).Build()
	c3Offline := clients.New(t).DisconnectedDuration(5 * time.Minute).Build()
	c4Offline := clients.New(t).DisconnectedDuration(time.Minute).Build()

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
			clientService := NewClientService(nil, nil, clients.NewClientRepository([]*clients.Client{c1Active, c2Active, c3Offline, c4Offline}, &hour, testLog))
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
	srv := ClientService{
		portDistributor: ports.NewPortDistributorForTests(
			mapset.NewThreadUnsafeSetFromSlice([]interface{}{1, 2, 3, 4, 5}),
			mapset.NewThreadUnsafeSetFromSlice([]interface{}{2, 3, 4}),
		),
	}

	invalidPort := "24563a"
	_, invalidPortParseErr := strconv.Atoi(invalidPort)

	testCases := []struct {
		name      string
		port      string
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
			name: "busy port",
			port: "5",
			wantError: errors2.APIError{
				Message:    "Local port 5 already in use.",
				HTTPStatus: http.StatusConflict,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotErr := srv.checkLocalPort(tc.port)

			// then
			require.Equal(t, tc.wantError, gotErr)
		})
	}
}

func TestCheckClientsAccess(t *testing.T) {
	c1 := clients.New(t).Build()                                                             // no groups
	c2 := clients.New(t).AllowedUserGroups([]string{users.Administrators}).Build()           // admin
	c3 := clients.New(t).AllowedUserGroups([]string{users.Administrators, "group1"}).Build() // admin + group1
	c4 := clients.New(t).AllowedUserGroups([]string{"group1"}).Build()                       // group1
	c5 := clients.New(t).AllowedUserGroups([]string{"group1", "group2"}).Build()             // group1 + group2
	c6 := clients.New(t).AllowedUserGroups([]string{"group3"}).Build()                       // group3

	allClients := []*clients.Client{c1, c2, c3, c4, c5, c6}
	testCases := []struct {
		name                      string
		clients                   []*clients.Client
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
			clients:                   []*clients.Client{c3, c4, c5, c6},
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			clientService := NewClientService(nil, nil, clients.NewClientRepository(allClients, nil, testLog))

			// when
			gotErr := clientService.CheckClientsAccess(tc.clients, tc.user)

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
