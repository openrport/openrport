package chshare

import (
	"testing"

	"github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
)

type MockService struct {
	service.Service
	Calls      []string
	MockStatus service.Status
}

func (s *MockService) Start() error {
	s.Calls = append(s.Calls, "Start")
	return nil
}
func (s *MockService) Stop() error {
	s.Calls = append(s.Calls, "Stop")
	return nil
}
func (s *MockService) Install() error {
	s.Calls = append(s.Calls, "Install")
	return nil
}
func (s *MockService) Uninstall() error {
	s.Calls = append(s.Calls, "Uninstall")
	return nil
}
func (s *MockService) Status() (service.Status, error) {
	return s.MockStatus, nil
}

func TestHandleServiceCommand(t *testing.T) {
	testCases := []struct {
		Name          string
		Command       string
		Status        service.Status
		ExpectedCalls []string
	}{
		{
			Name:          "Start",
			Command:       "start",
			Status:        service.StatusStopped,
			ExpectedCalls: []string{"Start"},
		}, {
			Name:          "Stop",
			Command:       "stop",
			Status:        service.StatusRunning,
			ExpectedCalls: []string{"Stop"},
		}, {
			Name:          "Install",
			Command:       "install",
			Status:        service.StatusStopped,
			ExpectedCalls: []string{"Install"},
		}, {
			Name:          "Uninstall",
			Command:       "uninstall",
			Status:        service.StatusStopped,
			ExpectedCalls: []string{"Uninstall"},
		}, {
			Name:          "Uninstall running",
			Command:       "uninstall",
			Status:        service.StatusRunning,
			ExpectedCalls: []string{"Stop", "Uninstall"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			mockService := &MockService{
				MockStatus: tc.Status,
			}

			if err := HandleServiceCommand(mockService, tc.Command); err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.ExpectedCalls, mockService.Calls)
		})
	}

}
