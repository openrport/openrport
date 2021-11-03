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
)

var testLog = chshare.NewLogger("monitoring", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)
var measurementInterval = time.Second * 60
var measurement1 = time.Date(2021, time.September, 1, 0, 0, 0, 0, time.Local)
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
	compare := measurement3.Format(layoutDb)
	require.Equal(t, compare, mC1.Timestamp)
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
	require.Equal(t, `{[{"pid":30212, "parent_pid": 4711, "name": "cinnamon"}]}`, pC1.Processes)
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
	pC1, err := dbProvider.GetProcessesNearestByClientID(ctx, "test_client_1", m2)
	require.NoError(t, err)
	require.NotNil(t, pC1)
	require.Equal(t, `{[{"pid":30211, "parent_pid": 4711, "name": "idea"}]}`, pC1.Processes)
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
	require.Equal(t, `{"free_b./":54182758400,"free_b./home":328029413376,"total_b./":105555197952,"total_b./home":364015185920}`, mC1.Mountpoints)
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
	mC1, err := dbProvider.GetMountpointsNearestByClientID(ctx, "test_client_1", m2)
	require.NoError(t, err)
	require.NotNil(t, mC1)
	require.Equal(t, `{"free_b./":44182758400,"free_b./home":228029413376,"total_b./":105555197952,"total_b./home":364015185920}`, mC1.Mountpoints)
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
