package monitoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	monitoring_api "github.com/cloudradar-monitoring/rport/server/api/monitoring"
	"github.com/cloudradar-monitoring/rport/share/query"
)

func TestMonitoringService_GetClientMetricsLatest(t *testing.T) {
	dbProvider := &DBProviderMock{
		MetricsPayload: &monitoring_api.ClientMetricsPayload{
			Timestamp:          time.Time{},
			CPUUsagePercent:    monitoring_api.CPUUsagePercent{},
			MemoryUsagePercent: monitoring_api.MemoryUsagePercent{},
			IOUsagePercent:     monitoring_api.IOUsagePercent{},
		},
		MetricsListPayload: nil,
		ProcessesPayload:   nil,
		MountpointsPayload: nil,
	}

	ctx := context.Background()

	service := NewService(dbProvider)
	clientID := "test_client"
	url := filepath.Join("/clients", clientID, "/metrics")
	req := httptest.NewRequest(http.MethodGet, url, nil)

	lo := query.NewOptions(req, monitoring_api.ClientMetricsSortDefault, monitoring_api.ClientMetricsFilterDefault, monitoring_api.ClientMetricsFieldsDefault)
	payload, err := service.GetClientMetricsLatest(ctx, clientID, lo)
	require.NoError(t, err)
	require.NotNil(t, payload)
}
