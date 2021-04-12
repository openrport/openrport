package chserver

import (
	"context"
	"errors"
	"net"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"

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
