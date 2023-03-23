package updates

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/logger"
	chshare "github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
)

type mockPackageManager struct {
	isAvailable bool
	status      *models.UpdatesStatus
	err         error
}

func (pm *mockPackageManager) IsAvailable(context.Context) bool {
	return pm.isAvailable
}

func (pm *mockPackageManager) GetUpdatesStatus(context.Context, *logger.Logger) (*models.UpdatesStatus, error) {
	newStatus := &models.UpdatesStatus{}
	if pm.status != nil {
		*newStatus = *pm.status
	}
	return newStatus, pm.err
}

type mockSSHRequest struct {
	Name string
	Data []byte
}

type mockSSHConn struct {
	ssh.Conn

	requests chan mockSSHRequest
}

func (c *mockSSHConn) SendRequest(name string, _ bool, data []byte) (bool, []byte, error) {
	c.requests <- mockSSHRequest{
		Name: name,
		Data: data,
	}

	return false, nil, nil
}

func TestUpdates(t *testing.T) {
	logger := chshare.NewLogger("test", chshare.NewLogOutput(""), chshare.LogLevelDebug)

	testCases := []struct {
		Name                     string
		Interval                 time.Duration
		NotAvailable             bool
		Status                   *models.UpdatesStatus
		PackageManagerErr        error
		NumRequests              int
		CallRefresh              bool
		ExpectedError            string
		ExpectedUpdatesAvailable int
	}{
		{
			Name:          "No available package manager",
			NotAvailable:  true,
			ExpectedError: "no supported package manager found",
		},
		{
			Name: "Send update on first status if connection is set",
			Status: &models.UpdatesStatus{
				UpdatesAvailable: 13,
			},
			ExpectedUpdatesAvailable: 13,
		},
		{
			Name: "Send update after connection is set",
			Status: &models.UpdatesStatus{
				UpdatesAvailable: 13,
			},
			ExpectedUpdatesAvailable: 13,
		},
		{
			Name:              "Package manager error",
			PackageManagerErr: errors.New("some error"),
			ExpectedError:     "some error",
		},
		{
			Name:     "Send update every interval",
			Interval: time.Millisecond,
			Status: &models.UpdatesStatus{
				UpdatesAvailable: 13,
			},
			NumRequests:              3,
			ExpectedUpdatesAvailable: 13,
		},
		{
			Name: "Send update after refresh",
			Status: &models.UpdatesStatus{
				UpdatesAvailable: 13,
			},
			CallRefresh:              true,
			NumRequests:              2,
			ExpectedUpdatesAvailable: 13,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			pm := &mockPackageManager{
				err:         tc.PackageManagerErr,
				status:      tc.Status,
				isAvailable: !tc.NotAvailable,
			}
			packageManagers = []PackageManager{pm}

			if tc.Interval == 0 {
				tc.Interval = time.Hour
			}
			updates := New(logger, tc.Interval)
			updates.Start(ctx)

			mockConn := &mockSSHConn{
				requests: make(chan mockSSHRequest, tc.NumRequests),
			}
			updates.SetConn(mockConn)

			if tc.NumRequests == 0 {
				tc.NumRequests = 1
			}
			for i := 0; i < tc.NumRequests; i++ {
				request := <-mockConn.requests

				var result models.UpdatesStatus
				err := json.Unmarshal(request.Data, &result)
				require.NoError(t, err)

				assert.Equal(t, comm.RequestTypeUpdatesStatus, request.Name)
				assert.Equal(t, tc.ExpectedError, result.Error)
				assert.Equal(t, tc.ExpectedUpdatesAvailable, result.UpdatesAvailable)
				assert.WithinDuration(t, time.Now(), result.Refreshed, 180*time.Millisecond)

				if tc.CallRefresh {
					updates.Refresh()
				}
			}
		})
	}
}
