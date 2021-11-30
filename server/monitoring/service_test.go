package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

func TestMonitoringService_SaveMeasurement(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	service := NewService(dbProvider)
	mClient := time.Now().UTC()
	m := &models.Measurement{
		ClientID:  "test1",
		Timestamp: mClient,
	}
	sleep := time.Duration(1) * time.Second
	time.Sleep(sleep)
	err = service.SaveMeasurement(context.Background(), m)
	require.NoError(t, err)
	gap := m.Timestamp.Sub(mClient)
	require.True(t, gap >= sleep, "monitoring.service must set timestamp")
}

func TestMonitoringService_ListClientMetrics(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
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
			require.Equal(t, tc.ExpectedMetaCount, payload.Meta.Count)

			metricsList, ok := payload.Data.([]*ClientMetricsPayload)
			require.True(t, ok)
			require.Equal(t, tc.ExpectedDataListLen, len(metricsList))
			require.Equal(t, tc.ExpectedTimestamp, metricsList[0].Timestamp)
		})
	}

}
