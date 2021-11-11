package monitoring

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/api/monitoring"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
	"github.com/cloudradar-monitoring/rport/share/types"
)

var testLog = chshare.NewLogger("monitoring", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)
var measurementInterval = time.Second * 60
var measurement1 = time.Date(2021, time.September, 1, 0, 0, 0, 0, time.UTC)
var measurement2 = measurement1.Add(measurementInterval)
var measurement3 = measurement2.Add(measurementInterval)
var testStart = time.Now()

var testData = []models.Measurement{
	{
		ClientID:           "test_client_1",
		Timestamp:          measurement1,
		CPUUsagePercent:    10,
		MemoryUsagePercent: 30,
		IoUsagePercent:     2,
		Processes:          `{[{"pid":30210, "parent_pid": 4711, "name": "chrome"}]}`,
		Mountpoints:        `{"free_b./":34182758400,"free_b./home":128029413376,"total_b./":105555197952,"total_b./home":364015185920}`,
	},
	{
		ClientID:           "test_client_1",
		Timestamp:          measurement2,
		CPUUsagePercent:    15,
		MemoryUsagePercent: 35,
		IoUsagePercent:     3,
		Processes:          `{[{"pid":30211, "parent_pid": 4711, "name": "idea"}]}`,
		Mountpoints:        `{"free_b./":44182758400,"free_b./home":228029413376,"total_b./":105555197952,"total_b./home":364015185920}`,
	},
	{
		ClientID:           "test_client_1",
		Timestamp:          measurement3,
		CPUUsagePercent:    20,
		MemoryUsagePercent: 40,
		IoUsagePercent:     4,
		Processes:          `{[{"pid":30212, "parent_pid": 4711, "name": "cinnamon"}]}`,
		Mountpoints:        `{"free_b./":54182758400,"free_b./home":328029413376,"total_b./":105555197952,"total_b./home":364015185920}`,
	},
}

func TestSqliteProvider_CreateMeasurement(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	m2 := &models.Measurement{
		ClientID:           "test_client_2",
		Timestamp:          testStart,
		CPUUsagePercent:    0,
		MemoryUsagePercent: 0,
		IoUsagePercent:     0,
		Processes:          `{[{"pid":30000, "parent_pid": 4712, "name": "firefox"}]}`,
		Mountpoints:        "{}",
	}
	// create new measurement
	err = dbProvider.CreateMeasurement(ctx, m2)
	require.NoError(t, err)
}

func TestSqliteProvider_DeleteMeasurementsBefore(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	deleted, err := dbProvider.DeleteMeasurementsBefore(ctx, measurement3)
	require.NoError(t, err)
	require.Equal(t, int64(2), deleted)
}

func TestSqliteProvider_CountByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get the latest metrics measurement of client
	options := createMetricsDefaultOptions()
	count, err := dbProvider.CountByClientID(ctx, "test_client_1", options)
	require.NoError(t, err)
	require.Equal(t, 3, count)
}

func TestSqliteProvider_ListMetricsLatestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get the latest metrics measurement of client
	options := createMetricsDefaultOptions()
	lm, err := dbProvider.ListMetricsByClientID(ctx, "test_client_1", options)
	require.NoError(t, err)
	require.NotNil(t, lm)
	require.Equal(t, 1, len(lm))
	require.Equal(t, measurement3, lm[0].Timestamp)
}

func TestSqliteProvider_ListMetricsNextByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get the second page, which is the second entry on page limit=1)
	options := createMetricsDefaultOptions()
	options.Pagination.Offset = "1"
	lm, err := dbProvider.ListMetricsByClientID(ctx, "test_client_1", options)
	require.NoError(t, err)
	require.NotNil(t, lm)
	require.Equal(t, 1, len(lm))
	require.Equal(t, measurement2, lm[0].Timestamp)
}

func TestSqliteProvider_ListGraphMetricsByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createDownsamplingData(ctx, dbProvider)
	require.NoError(t, err)

	options := createGraphMetricsDefaultOptions()
	hours := 48.0
	options.Filters = createSinceUntilFilter(measurement1, hours)

	mList, err := dbProvider.ListGraphMetricsByClientID(ctx, "test_client", hours, options)
	require.NoError(t, err)
	require.NotNil(t, mList)
	require.Equal(t, 126, len(mList))
}

func TestSqliteProvider_ListProcessesLatestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get the latest processes of client
	options := createProcessesDefaultOptions()
	pC1, err := dbProvider.ListProcessesByClientID(ctx, "test_client_1", options)
	require.NoError(t, err)
	require.NotNil(t, pC1)
	require.Equal(t, 1, len(pC1))
	var expectedJSON types.JSONString = `{[{"pid":30212, "parent_pid": 4711, "name": "cinnamon"}]}`
	require.Equal(t, expectedJSON, pC1[0].Processes)
}

func TestSqliteProvider_ListProcessesNextByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get the third page, which is the third entry on page limit=1)
	options := createProcessesDefaultOptions()
	options.Pagination.Offset = "2"

	pC1, err := dbProvider.ListProcessesByClientID(ctx, "test_client_1", options)
	require.NoError(t, err)
	require.NotNil(t, pC1)
	require.Equal(t, 1, len(pC1))
	var expectedJSON types.JSONString = `{[{"pid":30210, "parent_pid": 4711, "name": "chrome"}]}`
	require.Equal(t, expectedJSON, pC1[0].Processes)
}

func TestSqliteProvider_ListMountpointsLatestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get the latest mountpoints of client
	options := createMountpointsDefaultOptions()
	mC1, err := dbProvider.ListMountpointsByClientID(ctx, "test_client_1", options)
	require.NoError(t, err)
	require.NotNil(t, mC1)
	require.Equal(t, 1, len(mC1))
	var expectedJSON types.JSONString = `{"free_b./":54182758400,"free_b./home":328029413376,"total_b./":105555197952,"total_b./home":364015185920}`
	require.Equal(t, expectedJSON, mC1[0].Mountpoints)
}

func TestSqliteProvider_ListMountpointsPageByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get mountpoints of client with timestamp
	options := createMountpointsDefaultOptions()
	options.Pagination.Limit = "2"
	mC1, err := dbProvider.ListMountpointsByClientID(ctx, "test_client_1", options)
	require.NoError(t, err)
	require.NotNil(t, mC1)
	require.Equal(t, 2, len(mC1))
	var expectedJSON0 types.JSONString = `{"free_b./":54182758400,"free_b./home":328029413376,"total_b./":105555197952,"total_b./home":364015185920}`
	var expectedJSON1 types.JSONString = `{"free_b./":44182758400,"free_b./home":228029413376,"total_b./":105555197952,"total_b./home":364015185920}`
	require.Equal(t, expectedJSON0, mC1[0].Mountpoints)
	require.Equal(t, expectedJSON1, mC1[1].Mountpoints)
}

func createTestData(ctx context.Context, dbProvider DBProvider) error {
	for i := range testData {
		m := &models.Measurement{
			ClientID:           testData[i].ClientID,
			Timestamp:          testData[i].Timestamp,
			CPUUsagePercent:    testData[i].CPUUsagePercent,
			MemoryUsagePercent: testData[i].MemoryUsagePercent,
			IoUsagePercent:     testData[i].IoUsagePercent,
			Processes:          testData[i].Processes,
			Mountpoints:        testData[i].Mountpoints,
		}
		if err := dbProvider.CreateMeasurement(ctx, m); err != nil {
			return err
		}
	}

	return nil
}

func createDownsamplingData(ctx context.Context, dbProvider DBProvider) error {
	count := 60 * 48
	start := measurement1
	for i := 0; i < count; i++ {
		stamp := start.Add(time.Duration(i) * measurementInterval)
		r := float64(i % 2)
		m := &models.Measurement{
			ClientID:           "test_client",
			Timestamp:          stamp,
			CPUUsagePercent:    10.0 + r*10,
			MemoryUsagePercent: 10.0 + r*10,
			IoUsagePercent:     10.0 + r*10,
			Processes:          "",
			Mountpoints:        "",
		}
		if err := dbProvider.CreateMeasurement(ctx, m); err != nil {
			return err
		}
	}

	return nil
}

func createSinceUntilFilter(start time.Time, hours float64) []query.FilterOption {
	mSince := start.Format(layoutDb)
	tUntil := start.Add(time.Duration(hours) * time.Hour)
	mUntil := tUntil.Format(layoutDb)

	filters := []query.FilterOption{
		{
			Expression: "timestamp[since]",
			Column:     "timestamp",
			Operator:   query.FilterOperatorTypeGT,
			Values:     []string{mSince},
		},
		{
			Expression: "timestamp[until]",
			Column:     "timestamp",
			Operator:   query.FilterOperatorTypeLT,
			Values:     []string{mUntil},
		},
	}

	return filters
}

func createGraphMetricsDefaultOptions() *query.ListOptions {
	qOptions := &query.ListOptions{}

	qOptions.Sorts = query.ParseSortOptions(monitoring.ClientGraphMetricsSortDefault)
	qOptions.Filters = query.ParseFilterOptions(monitoring.ClientGraphMetricsFilterDefault)
	qOptions.Fields = query.ParseFieldsOptions(monitoring.ClientGraphMetricsFieldsDefault)

	return qOptions
}

func createMetricsDefaultOptions() *query.ListOptions {
	qOptions := &query.ListOptions{}

	qOptions.Sorts = query.ParseSortOptions(monitoring.ClientMetricsSortDefault)
	qOptions.Filters = query.ParseFilterOptions(monitoring.ClientMetricsFilterDefault)
	qOptions.Fields = query.ParseFieldsOptions(monitoring.ClientMetricsFieldsDefault)
	qOptions.Pagination = &query.Pagination{
		Limit:  "1",
		Offset: "0",
	}

	return qOptions
}

func createProcessesDefaultOptions() *query.ListOptions {
	qOptions := &query.ListOptions{}

	qOptions.Sorts = query.ParseSortOptions(monitoring.ClientProcessesSortDefault)
	qOptions.Filters = query.ParseFilterOptions(monitoring.ClientProcessesFilterDefault)
	qOptions.Fields = query.ParseFieldsOptions(monitoring.ClientProcessesFieldsDefault)
	qOptions.Pagination = &query.Pagination{
		Limit:  "1",
		Offset: "0",
	}

	return qOptions
}

func createMountpointsDefaultOptions() *query.ListOptions {
	qOptions := &query.ListOptions{}

	qOptions.Sorts = query.ParseSortOptions(monitoring.ClientMountpointsSortDefault)
	qOptions.Filters = query.ParseFilterOptions(monitoring.ClientMountpointsFilterDefault)
	qOptions.Fields = query.ParseFieldsOptions(monitoring.ClientMountpointsFieldsDefault)
	qOptions.Pagination = &query.Pagination{
		Limit:  "1",
		Offset: "0",
	}

	return qOptions
}
