package monitoring

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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

func TestSqliteProvider_GetMetricsLatestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get the latest metrics measurement of client
	fields := make([]query.FieldsOption, 1)
	fields[0] = query.FieldsOption{
		Resource: "metrics",
		Fields:   []string{"timestamp", "cpu_usage_percent", "memory_usage_percent"},
	}
	mC1, err := dbProvider.GetMetricsLatestByClientID(ctx, "test_client_1", fields)
	require.NoError(t, err)
	require.NotNil(t, mC1)
	require.Equal(t, measurement3, mC1.Timestamp)
}

func TestSqliteProvider_GetMetricsListByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	sorts := []query.SortOption{query.SortOption{
		Column: "timestamp",
		IsASC:  false,
	},
	}
	fields := make([]query.FieldsOption, 1)
	fields[0] = query.FieldsOption{
		Resource: "metrics",
		Fields:   []string{"timestamp", "cpu_usage_percent", "memory_usage_percent"},
	}
	ts1 := measurement1.Format(layoutDb)
	filters := []query.FilterOption{
		{
			Expression: "timestamp[gt]",
			Column:     "timestamp",
			Operator:   query.FilterOperatorTypeGT,
			Values:     []string{ts1},
		},
		{
			Expression: "limit",
			Column:     "limit",
			Operator:   query.FilterOperatorTypeEQ,
			Values:     []string{"2"},
		},
	}
	lo := &query.ListOptions{
		Sorts:   sorts,
		Filters: filters,
		Fields:  fields,
	}
	mList, err := dbProvider.GetMetricsListByClientID(ctx, "test_client_1", lo)
	require.NoError(t, err)
	require.NotNil(t, mList)
	require.Equal(t, 2, len(mList))
}

func TestSqliteProvider_GetMetricsListDownsampledByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createDownsamplingData(ctx, dbProvider)
	require.NoError(t, err)

	sorts := []query.SortOption{query.SortOption{
		Column: "timestamp",
		IsASC:  false,
	},
	}
	fields := make([]query.FieldsOption, 1)
	fields[0] = query.FieldsOption{
		Resource: "metrics",
		Fields:   []string{"timestamp", "cpu_usage_percent", "memory_usage_percent"},
	}
	hours := 48.0
	lo := &query.ListOptions{
		Sorts:   sorts,
		Filters: createSinceUntilFilter(measurement1, hours),
		Fields:  fields,
	}
	mList, err := dbProvider.GetMetricsListDownsampledByClientID(ctx, "test_client", hours, lo)
	require.NoError(t, err)
	require.NotNil(t, mList)
	require.Equal(t, 126, len(mList))
}

func TestSqliteProvider_GetProcessesLatestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get the latest processes of client
	pC1, err := dbProvider.GetProcessesLatestByClientID(ctx, "test_client_1")
	require.NoError(t, err)
	require.NotNil(t, pC1)
	var expectedJSON types.JSONString = `{[{"pid":30212, "parent_pid": 4711, "name": "cinnamon"}]}`
	require.Equal(t, expectedJSON, pC1.Processes)
}

func TestSqliteProvider_GetProcessesNearestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get processes of client with timestamp
	m2 := measurement1.Add(measurementInterval)
	pC1, err := dbProvider.GetProcessesNearestByClientID(ctx, "test_client_1", createSinceFilter(m2))
	require.NoError(t, err)
	require.NotNil(t, pC1)
	var expectedJSON types.JSONString = `{[{"pid":30211, "parent_pid": 4711, "name": "idea"}]}`
	require.Equal(t, expectedJSON, pC1.Processes)
}

func TestSqliteProvider_GetMountpointsLatestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get the latest mountpoints of client
	mC1, err := dbProvider.GetMountpointsLatestByClientID(ctx, "test_client_1")
	require.NoError(t, err)
	require.NotNil(t, mC1)
	var expectedJSON types.JSONString = `{"free_b./":54182758400,"free_b./home":328029413376,"total_b./":105555197952,"total_b./home":364015185920}`
	require.Equal(t, expectedJSON, mC1.Mountpoints)
}

func TestSqliteProvider_GetMountpointsNearestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(ctx, dbProvider)
	require.NoError(t, err)

	// get mountpoints of client with timestamp
	m2 := measurement1.Add(measurementInterval)
	mC1, err := dbProvider.GetMountpointsNearestByClientID(ctx, "test_client_1", createSinceFilter(m2))
	require.NoError(t, err)
	require.NotNil(t, mC1)
	var expectedJSON types.JSONString = `{"free_b./":44182758400,"free_b./home":228029413376,"total_b./":105555197952,"total_b./home":364015185920}`
	require.Equal(t, expectedJSON, mC1.Mountpoints)
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

func createSinceFilter(since time.Time) []query.FilterOption {
	mSince := since.Format(layoutDb)

	filters := []query.FilterOption{
		{
			Expression: "timestamp[since]",
			Column:     "timestamp",
			Operator:   query.FilterOperatorTypeSince,
			Values:     []string{mSince},
		},
	}

	return filters
}
