package chserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/server/clients"
	"github.com/realvnc-labs/rport/server/clients/clientdata"
	"github.com/realvnc-labs/rport/server/monitoring"
	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/test"
)

func TestHandleRefreshUpdatesStatus(t *testing.T) {
	c1 := clients.New(t).Logger(testLog).Build()
	c2 := clients.New(t).DisconnectedDuration(5 * time.Minute).Logger(testLog).Build()

	testCases := []struct {
		Name                string
		ClientID            string
		SSHError            bool
		ExpectedStatus      int
		ExpectedRequestName string
	}{
		{
			Name:                "Connected client",
			ClientID:            c1.GetID(),
			ExpectedStatus:      http.StatusNoContent,
			ExpectedRequestName: comm.RequestTypeRefreshUpdatesStatus,
		},
		{
			Name:           "Disconnected client",
			ClientID:       c2.GetID(),
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "Non-existing client",
			ClientID:       "non-existing-client",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:                "SSH error",
			ClientID:            c1.GetID(),
			SSHError:            true,
			ExpectedRequestName: comm.RequestTypeRefreshUpdatesStatus,
			ExpectedStatus:      http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			connMock := test.NewConnMock()
			// by default set to return success
			connMock.ReturnOk = !tc.SSHError
			c1.SetConnection(connMock)
			clientService := clients.NewClientService(nil, nil, clients.NewClientRepository([]*clientdata.Client{c1, c2}, &hour, testLog), testLog, nil)
			al := APIListener{
				insecureForTests: true,
				Server: &Server{
					clientService: clientService,
					config:        &chconfig.Config{},
				},
				Logger: testLog,
			}
			al.initRouter()

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clients/%s/updates-status", tc.ClientID), nil)

			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			assert.Equal(t, tc.ExpectedStatus, w.Code)
			if tc.ExpectedRequestName != "" {
				name, _, _ := connMock.InputSendRequest()
				assert.Equal(t, tc.ExpectedRequestName, name)
			}
		})
	}
}

func TestListClientMetrics(t *testing.T) {
	m1 := time.Date(2021, time.September, 1, 0, 0, 0, 0, time.UTC)
	m2 := time.Date(2021, time.September, 1, 0, 1, 0, 0, time.UTC)
	cmp1 := &monitoring.ClientMetricsPayload{
		Timestamp:          m1,
		CPUUsagePercent:    10.5,
		MemoryUsagePercent: 2.5,
		IOUsagePercent:     20,
	}
	cmp2 := &monitoring.ClientMetricsPayload{
		Timestamp:          m2,
		CPUUsagePercent:    20.5,
		MemoryUsagePercent: 2.5,
		IOUsagePercent:     25,
	}
	lcmp := []*monitoring.ClientMetricsPayload{cmp1, cmp2}

	cpp1 := &monitoring.ClientProcessesPayload{
		Timestamp: m1,
		Processes: `[{"pid":30212,"parent_pid":4711,"name":"chrome"}]`,
	}
	lcpp := []*monitoring.ClientProcessesPayload{cpp1}
	dbProvider := &monitoring.DBProviderMock{
		MetricsListPayload:     lcmp,
		ProcessesListPayload:   lcpp,
		MountpointsListPayload: nil,
	}
	monitoringService := monitoring.NewService(dbProvider, testLog)
	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			config: &chconfig.Config{
				Monitoring: chconfig.MonitoringConfig{
					Enabled: true,
				},
			},
			monitoringService: monitoringService,
		},
	}
	al.initRouter()

	testCases := []struct {
		Name           string
		URL            string
		ExpectedStatus int
		ExpectedJSON   string
	}{
		{
			Name:           "metrics default, no filter, no fields",
			URL:            "metrics",
			ExpectedStatus: http.StatusOK,
			ExpectedJSON:   `{"data":[{"timestamp":"2021-09-01T00:00:00Z","cpu_usage_percent":10.5,"memory_usage_percent":2.5,"io_usage_percent":20},{"timestamp":"2021-09-01T00:01:00Z","cpu_usage_percent":20.5,"memory_usage_percent":2.5,"io_usage_percent":25}],"meta":{"count":10}}`,
		},
		{
			Name:           "metrics with fields, no filter, unknown field",
			URL:            "metrics?fields[metrics]=timestamp,cpu_usage_percent,unknown_field",
			ExpectedStatus: http.StatusBadRequest,
			ExpectedJSON:   `{"errors":[{"code":"","title":"unsupported field \"unknown_field\" for resource \"metrics\"","detail":""}]}`,
		},
		{
			Name:           "metrics with timestamp filter, filter ok",
			URL:            "metrics?filter[timestamp][gt]=1636009200&filter[timestamp][lt]=1636012800",
			ExpectedStatus: http.StatusOK,
			ExpectedJSON:   `{"data":[{"timestamp":"2021-09-01T00:00:00Z","cpu_usage_percent":10.5,"memory_usage_percent":2.5,"io_usage_percent":20},{"timestamp":"2021-09-01T00:01:00Z","cpu_usage_percent":20.5,"memory_usage_percent":2.5,"io_usage_percent":25}],"meta":{"count":10}}`,
		},
		{
			Name:           "metrics with datetime filter, filter ok",
			URL:            "metrics?filter[timestamp][since]=2021-09-01T00:00:00%2B00:00&filter[timestamp][until]=2021-09-01T00:01:00%2B00:00",
			ExpectedStatus: http.StatusOK,
			ExpectedJSON:   `{"data":[{"timestamp":"2021-09-01T00:00:00Z","cpu_usage_percent":10.5,"memory_usage_percent":2.5,"io_usage_percent":20},{"timestamp":"2021-09-01T00:01:00Z","cpu_usage_percent":20.5,"memory_usage_percent":2.5,"io_usage_percent":25}],"meta":{"count":10}}`,
		},
		{
			Name:           "processes default, no filter, no fields",
			URL:            "processes",
			ExpectedStatus: http.StatusOK,
			ExpectedJSON:   `{"data":[{"timestamp":"2021-09-01T00:00:00Z","processes":[{"pid":30212,"parent_pid":4711,"name":"chrome"}]}],"meta":{"count":10}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/clients/test_client/"+tc.URL, nil)
			al.router.ServeHTTP(w, req)

			assert.Equal(t, tc.ExpectedStatus, w.Code)

			gotJSON := w.Body.String()
			assert.JSONEq(t, tc.ExpectedJSON, gotJSON)
		})
	}
}

func TestMonitoringDisabled(t *testing.T) {
	m1 := time.Date(2021, time.September, 1, 0, 0, 0, 0, time.UTC)
	m2 := time.Date(2021, time.September, 1, 0, 1, 0, 0, time.UTC)
	cmp1 := &monitoring.ClientMetricsPayload{
		Timestamp:          m1,
		CPUUsagePercent:    10.5,
		MemoryUsagePercent: 2.5,
		IOUsagePercent:     20,
	}
	cmp2 := &monitoring.ClientMetricsPayload{
		Timestamp:          m2,
		CPUUsagePercent:    20.5,
		MemoryUsagePercent: 2.5,
		IOUsagePercent:     25,
	}
	lcmp := []*monitoring.ClientMetricsPayload{cmp1, cmp2}

	cpp1 := &monitoring.ClientProcessesPayload{
		Timestamp: m1,
		Processes: `[{"pid":30212,"parent_pid":4711,"name":"chrome"}]`,
	}
	lcpp := []*monitoring.ClientProcessesPayload{cpp1}
	dbProvider := &monitoring.DBProviderMock{
		MetricsListPayload:     lcmp,
		ProcessesListPayload:   lcpp,
		MountpointsListPayload: nil,
	}

	monitoringService := monitoring.NewService(dbProvider, testLog)

	testCases := []struct {
		Name           string
		URL            string
		Enabled        bool
		ExpectedStatus int
	}{
		{
			Name:           "metrics, monitoring enabled",
			URL:            "metrics",
			Enabled:        true,
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "metrics, monitoring disabled",
			URL:            "metrics",
			Enabled:        false,
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "processes, monitoring enabled",
			URL:            "processes",
			Enabled:        true,
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "processes, monitoring disabled",
			URL:            "processes",
			Enabled:        false,
			ExpectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			al := APIListener{
				insecureForTests: true,
				Server: &Server{
					config: &chconfig.Config{
						Monitoring: chconfig.MonitoringConfig{
							Enabled: tc.Enabled,
						},
					},
					monitoringService: monitoringService,
				},
			}
			al.initRouter()

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/clients/test_client/"+tc.URL, nil)
			al.router.ServeHTTP(w, req)

			assert.Equal(t, tc.ExpectedStatus, w.Code)
			if tc.ExpectedStatus != http.StatusOK {
				gotJSON := w.Body.String()
				assert.Contains(t, gotJSON, "monitoring disabled")
			}
		})
	}
}
