package monitoring

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"
)

var testLog = chshare.NewLogger("monitoring", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)
var measurementStart = time.Date(2021, time.September, 1, 0, 0, 0, 0, time.UTC).Unix()
var testStart = time.Now().Unix()
var measurementInterval int64 = 60

var testData = []models.Measurement{
	{
		ClientID:           "test_client_1",
		Timestamp:          measurementStart,
		CPUUsagePercent:    10,
		MemoryUsagePercent: 30,
		IoUsagePercent:     2,
		Processes:          `{[{"pid":30210, "parent_pid": 4711, "name": "chrome"}]}`,
		Mountpoints:        `{"free_b./":34182758400,"free_b./home":128029413376,"total_b./":105555197952,"total_b./home":364015185920}`,
	},
	{
		ClientID:           "test_client_1",
		Timestamp:          measurementStart + measurementInterval,
		CPUUsagePercent:    15,
		MemoryUsagePercent: 35,
		IoUsagePercent:     3,
		Processes:          `{[{"pid":30211, "parent_pid": 4711, "name": "idea"}]}`,
		Mountpoints:        `{"free_b./":44182758400,"free_b./home":228029413376,"total_b./":105555197952,"total_b./home":364015185920}`,
	},
	{
		ClientID:           "test_client_1",
		Timestamp:          measurementStart + measurementInterval + measurementInterval,
		CPUUsagePercent:    20,
		MemoryUsagePercent: 40,
		IoUsagePercent:     4,
		Processes:          `{[{"pid":30212, "parent_pid": 4711, "name": "cinnamon"}]}`,
		Mountpoints:        `{"free_b./":54182758400,"free_b./home":328029413376,"total_b./":105555197952,"total_b./home":364015185920}`,
	},
}

func TestDBProvider(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(dbProvider.DB())
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

	// get latest of client
	mC1, err := dbProvider.GetClientLatest(ctx, "test_client_1")
	require.NoError(t, err)
	require.NotNil(t, mC1)
	require.Equal(t, measurementStart+measurementInterval+measurementInterval, mC1.Timestamp)

	// delete old measurements (older than 30 days)
	compare := testStart - (30 * 3600)
	deleted, err := dbProvider.DeleteMeasurementsOlderThan(ctx, compare)
	require.NoError(t, err)
	require.Equal(t, int64(len(testData)), deleted)

	// delete all remaining measurements
	compare = testStart + 1
	deleted, err = dbProvider.DeleteMeasurementsOlderThan(ctx, compare)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
}

func TestSqliteProvider_GetProcessesLatestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(dbProvider.DB())
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

	err = createTestData(dbProvider.DB())
	require.NoError(t, err)

	// get processes of client with timestamp
	m2 := measurementStart + measurementInterval
	pC1, err := dbProvider.GetProcessesNearestByClientID(ctx, "test_client_1", strconv.FormatInt(m2, 10))
	require.NoError(t, err)
	require.NotNil(t, pC1)
	require.Equal(t, `{[{"pid":30211, "parent_pid": 4711, "name": "idea"}]}`, pC1.Processes)
}

func TestSqliteProvider_GetMountpointsLatestByClientID(t *testing.T) {
	dbProvider, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProvider.Close()

	ctx := context.Background()

	err = createTestData(dbProvider.DB())
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

	err = createTestData(dbProvider.DB())
	require.NoError(t, err)

	// get mountpoints of client with timestamp
	m2 := measurementStart + measurementInterval
	mC1, err := dbProvider.GetMountpointsNearestByClientID(ctx, "test_client_1", strconv.FormatInt(m2, 10))
	require.NoError(t, err)
	require.NotNil(t, mC1)
	require.Equal(t, `{"free_b./":44182758400,"free_b./home":228029413376,"total_b./":105555197952,"total_b./home":364015185920}`, mC1.Mountpoints)
}

func createTestData(db *sqlx.DB) error {
	for i := range testData {
		_, err := db.Exec(
			"INSERT INTO `measurements` (`client_id`, `timestamp`, `cpu_usage_percent`, `memory_usage_percent`, `io_usage_percent`, `processes`, `mountpoints`) VALUES (?,?,?,?,?,?,?)",
			testData[i].ClientID,
			testData[i].Timestamp,
			testData[i].CPUUsagePercent,
			testData[i].MemoryUsagePercent,
			testData[i].IoUsagePercent,
			testData[i].Processes,
			testData[i].Mountpoints,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
