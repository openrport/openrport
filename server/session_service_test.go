package chserver

import (
	"context"
	"errors"
	"net"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/test"
)

func TestStartClientSession(t *testing.T) {
	connMock := test.NewConnMock()
	connMock.ReturnRemoteAddr = &net.IPAddr{IP: net.IPv4(192, 0, 2, 1)}

	testCases := []struct {
		Name              string
		ClientAuthID      string
		SessionID         string
		AuthMultiuseCreds bool
		ExpectedError     error
	}{
		{
			Name:          "existing session id same client",
			ClientAuthID:  "test-client",
			SessionID:     "test-session",
			ExpectedError: errors.New("session id \"test-session\" is already in use"),
		}, {
			Name:          "existing session id different client",
			ClientAuthID:  "test-client-2",
			SessionID:     "test-session",
			ExpectedError: errors.New("session id \"test-session\" is already in use"),
		}, {
			Name:          "existing session with different id for client",
			ClientAuthID:  "test-client",
			SessionID:     "test-session-2",
			ExpectedError: errors.New("client auth ID is already in use: \"test-client\""),
		}, {
			Name:          "no existing session",
			ClientAuthID:  "test-client-2",
			SessionID:     "test-session-2",
			ExpectedError: nil,
		}, {
			Name:              "existing session id same client, auth multiuse",
			ClientAuthID:      "test-client",
			SessionID:         "test-session",
			AuthMultiuseCreds: true,
			ExpectedError:     errors.New("session id \"test-session\" is already in use"),
		}, {
			Name:              "existing session id different client, auth multiuse",
			ClientAuthID:      "test-client-2",
			SessionID:         "test-session",
			AuthMultiuseCreds: true,
			ExpectedError:     errors.New("session id \"test-session\" is already in use"),
		}, {
			Name:              "existing session with different id for client, auth multiuse",
			ClientAuthID:      "test-client",
			SessionID:         "test-session-2",
			AuthMultiuseCreds: true,
			ExpectedError:     nil,
		}, {
			Name:              "no existing session, auth multiuse",
			ClientAuthID:      "test-client-2",
			SessionID:         "test-session-2",
			AuthMultiuseCreds: true,
			ExpectedError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ss := &SessionService{
				repo: sessions.NewSessionRepository([]*sessions.ClientSession{{
					ID:           "test-session",
					ClientAuthID: "test-client",
				}}, nil),
				portDistributor: ports.NewPortDistributor(mapset.NewThreadUnsafeSet()),
			}
			_, err := ss.StartClientSession(
				context.Background(), tc.ClientAuthID, tc.SessionID, connMock, tc.AuthMultiuseCreds,
				&chshare.ConnectionRequest{}, testLog)
			assert.Equal(t, tc.ExpectedError, err)
		})
	}
}
