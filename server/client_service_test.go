package chserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
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
			ExpectedError: errors.New("client id \"test-client\" is already in use"),
		}, {
			Name:          "existing client id different client",
			ClientAuthID:  "test-client-auth-2",
			ClientID:      "test-client",
			ExpectedError: errors.New("client id \"test-client\" is already in use"),
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
			ExpectedError:     errors.New("client id \"test-client\" is already in use"),
		}, {
			Name:              "existing client id different client auth, auth multiuse",
			ClientAuthID:      "test-client-auth-2",
			ClientID:          "test-client",
			AuthMultiuseCreds: true,
			ExpectedError:     errors.New("client id \"test-client\" is already in use"),
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
				}}, nil),
				portDistributor: ports.NewPortDistributor(mapset.NewThreadUnsafeSet()),
			}
			_, err := cs.StartClient(
				context.Background(), tc.ClientAuthID, tc.ClientID, connMock, tc.AuthMultiuseCreds,
				&chshare.ConnectionRequest{}, testLog)
			assert.Equal(t, tc.ExpectedError, err)
		})
	}
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
				Message: "Client is active, should be disconnected",
				Code:    http.StatusBadRequest,
			},
		},
		{
			name:     "delete unknown client",
			clientID: "unknown-id",
			wantError: errors2.APIError{
				Message: "Client not found",
				Code:    http.StatusNotFound,
			},
		},
		{
			name:     "empty client ID",
			clientID: "",
			wantError: errors2.APIError{
				Message: "Client id is empty",
				Code:    http.StatusBadRequest,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			clientService := NewClientService(nil, clients.NewClientRepository([]*clients.Client{c1Active, c2Active, c3Offline, c4Offline}, &hour))
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
				Message: "Invalid local port: 24563a.",
				Err:     invalidPortParseErr,
				Code:    http.StatusBadRequest,
			},
		},
		{
			name: "not allowed port",
			port: "6",
			wantError: errors2.APIError{
				Message: "Local port 6 is not among allowed ports.",
				Code:    http.StatusBadRequest,
			},
		},
		{
			name: "busy port",
			port: "5",
			wantError: errors2.APIError{
				Message: "Local port 5 already in use.",
				Code:    http.StatusConflict,
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
