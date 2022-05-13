package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/db/sqlite"

	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}

func TestMonitoringService_SaveMeasurement(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", DataSourceOptions, testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	service := NewService(dbProvider)
	minGap := time.Second
	mClient := time.Now().UTC().Add(-minGap)
	m := &models.Measurement{
		ClientID:  "test1",
		Timestamp: mClient,
	}
	err = service.SaveMeasurement(context.Background(), m)
	require.NoError(t, err)
	gap := m.Timestamp.Sub(mClient)
	require.True(t, gap >= minGap, "monitoring.service must set timestamp")
}

func TestMonitoringService_ListClientMetrics(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", DataSourceOptions, testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	service := NewService(dbProvider)

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	options1 := createMetricsDefaultOptions()
	options2 := createMetricsDefaultOptions()
	options2.Sorts[0].IsASC = true
	options3 := createMetricsDefaultOptions()
	options3.Pagination.Limit = "2"

	testCases := []struct {
		Name                string
		Options             *query.ListOptions
		ExpectedMetaCount   int
		ExpectedDataListLen int
		ExpectedTimestamp   time.Time
	}{
		{
			Name:                "default, sort DESC",
			Options:             options1,
			ExpectedMetaCount:   3,
			ExpectedDataListLen: 1,
			ExpectedTimestamp:   measurement3,
		},
		{
			Name:                "default, sort ASC",
			Options:             options2,
			ExpectedMetaCount:   3,
			ExpectedDataListLen: 1,
			ExpectedTimestamp:   measurement1,
		},
		{
			Name:                "default, page[limit]=2",
			Options:             options3,
			ExpectedMetaCount:   3,
			ExpectedDataListLen: 2,
			ExpectedTimestamp:   measurement3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			payload, err := service.ListClientMetrics(ctx, "test_client_1", tc.Options)
			require.NoError(t, err)
			require.NotNil(t, payload.Data)
			require.NotNil(t, payload.Meta)
			require.Nil(t, payload.Links)
			require.Equal(t, tc.ExpectedMetaCount, payload.Meta.Count)

			metricsList, ok := payload.Data.([]*ClientMetricsPayload)
			require.True(t, ok)
			require.Equal(t, tc.ExpectedDataListLen, len(metricsList))
			require.Equal(t, tc.ExpectedTimestamp, metricsList[0].Timestamp)
		})
	}
}
func TestMonitoringService_ListClientGraphMetrics(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", DataSourceOptions, testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	service := NewService(dbProvider)

	ctx := context.Background()

	err = createDownsamplingData(ctx, dbProvider)
	require.NoError(t, err)

	hours := 48.0
	options1 := createGraphMetricsDefaultOptions(measurement1, hours, layoutAPI)
	options2 := createGraphMetricsDefaultOptions(measurement1, hours, layoutAPI)
	options3 := createGraphMetricsDefaultOptions(measurement1, hours, layoutAPI)

	url := "https://localhost:3000/api/v1/clients/graph-metrics"
	linkLanPercent := url + "/" + LinkNetPercentLan
	linkLanBPS := url + "/" + LinkNetBPSLan
	linkWanPercent := url + "/" + LinkNetPercentWan
	linkWanBPS := url + "/" + LinkNetBPSWan

	testCases := []struct {
		Name                   string
		Options                *query.ListOptions
		RequestInfo            *query.RequestInfo
		NetLan                 bool
		NetWan                 bool
		ExpectedLinksCount     int
		ExpectedDataListLen    int
		ExpectedLinkLanPercent *string
		ExpectedLinkLanBPS     *string
		ExpectedLinkWanPercent *string
		ExpectedLinkWanBPS     *string
	}{
		{
			Name:                   "links lan available",
			Options:                options1,
			RequestInfo:            &query.RequestInfo{URL: url},
			NetLan:                 true,
			NetWan:                 false,
			ExpectedDataListLen:    126,
			ExpectedLinkLanPercent: &linkLanPercent,
			ExpectedLinkLanBPS:     &linkLanBPS,
			ExpectedLinkWanPercent: nil,
			ExpectedLinkWanBPS:     nil,
		},
		{
			Name:                   "links lan and wan available",
			Options:                options2,
			RequestInfo:            &query.RequestInfo{URL: url},
			NetLan:                 true,
			NetWan:                 true,
			ExpectedDataListLen:    126,
			ExpectedLinkLanPercent: &linkLanPercent,
			ExpectedLinkLanBPS:     &linkLanBPS,
			ExpectedLinkWanPercent: &linkWanPercent,
			ExpectedLinkWanBPS:     &linkWanBPS,
		},
		{
			Name:                   "links lan and wan not available",
			Options:                options3,
			RequestInfo:            &query.RequestInfo{URL: "url"},
			NetLan:                 false,
			NetWan:                 false,
			ExpectedDataListLen:    126,
			ExpectedLinkLanPercent: nil,
			ExpectedLinkLanBPS:     nil,
			ExpectedLinkWanPercent: nil,
			ExpectedLinkWanBPS:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			payload, err := service.ListClientGraphMetrics(ctx, "test_client", tc.Options, tc.RequestInfo, tc.NetLan, tc.NetWan)
			require.NoError(t, err)
			require.NotNil(t, payload.Data)
			require.Nil(t, payload.Meta)
			require.NotNil(t, payload.Links)

			metricsList, ok := payload.Data.([]*ClientGraphMetricsPayload)
			require.True(t, ok)
			require.Equal(t, tc.ExpectedDataListLen, len(metricsList))

			links, ok := payload.Links.(*GraphMetricsLinksPayload)
			require.True(t, ok)
			require.Equal(t, tc.ExpectedLinkLanPercent, links.NetLanUsagePercent)
			require.Equal(t, tc.ExpectedLinkLanBPS, links.NetLanUsageBPS)
			require.Equal(t, tc.ExpectedLinkWanPercent, links.NetWanUsagePercent)
			require.Equal(t, tc.ExpectedLinkWanBPS, links.NetWanUsageBPS)
		})
	}

}
