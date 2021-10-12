package monitoring

import (
	"context"
	"os"
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
		Processes:          "{}",
		Mountpoints:        "{}",
	},
	{
		ClientID:           "test_client_1",
		Timestamp:          measurementStart + measurementInterval,
		CPUUsagePercent:    15,
		MemoryUsagePercent: 30,
		IoUsagePercent:     2,
		Processes:          "{}",
		Mountpoints:        "{}",
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
		Processes:          "{}",
		Mountpoints:        "{}",
	}
	// create new measurement
	err = dbProvider.CreateMeasurement(ctx, m2)
	require.NoError(t, err)

	// get latest of client
	mC1, err := dbProvider.GetClientLatest(ctx, "test_client_1")
	require.NoError(t, err)
	require.NotNil(t, mC1)
	require.Equal(t, measurementStart+measurementInterval, mC1.Timestamp)

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
