package jobs

import (
	"context"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/db/migration/jobs"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/server/test/jb"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}
var testLog = logger.NewLogger("api-listener-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

func TestJobsSqliteProvider(t *testing.T) {
	ctx := context.Background()
	jobsDB, err := sqlite.New(":memory:", jobs.AssetNames(), jobs.Asset, DataSourceOptions)
	require.NoError(t, err)
	p := NewSqliteProvider(jobsDB, testLog)
	defer p.Close()

	// add jobs
	job1 := jb.New(t).Status(models.JobStatusRunning).Result(nil).IsSudo().Build()
	job2 := jb.New(t).ClientID(job1.ClientID).Cwd("/root").Build()
	job3 := jb.New(t).Build() // different client ID
	require.NoError(t, p.SaveJob(job1))
	require.NoError(t, p.SaveJob(job2))
	require.NoError(t, p.SaveJob(job3))

	// verify added jobs
	gotJob1, err := p.GetByJID(job1.ClientID, job1.JID)
	require.NoError(t, err)
	require.NotNil(t, gotJob1)
	assert.Equal(t, job1, gotJob1)

	gotJob2, err := p.GetByJID(job2.ClientID, job2.JID)
	require.NoError(t, err)
	require.NotNil(t, gotJob2)
	assert.Equal(t, job2, gotJob2)

	gotJob3, err := p.GetByJID(job3.ClientID, job3.JID)
	require.NoError(t, err)
	require.NotNil(t, gotJob3)
	assert.Equal(t, job3, gotJob3)

	// verify not found job
	gotJob4, err := p.GetByJID(job3.ClientID, "unknown-jid")
	require.NoError(t, err)
	require.Nil(t, gotJob4)

	// verify job summaries
	gotJSc1, err := p.List(ctx, &query.ListOptions{Filters: []query.FilterOption{{Column: []string{"client_id"}, Values: []string{job1.ClientID}}}})
	require.NoError(t, err)
	assert.ElementsMatch(t, []*models.Job{job1, job2}, gotJSc1)

	gotJSc2, err := p.List(ctx, &query.ListOptions{Filters: []query.FilterOption{{Column: []string{"client_id"}, Values: []string{job3.ClientID}}}})
	require.NoError(t, err)
	assert.ElementsMatch(t, []*models.Job{job3}, gotJSc2)

	// verify job summaries not found
	gotJSc3, err := p.List(ctx, &query.ListOptions{Filters: []query.FilterOption{{Column: []string{"client_id"}, Values: []string{"unknown-cid"}}}})
	require.NoError(t, err)
	require.Empty(t, gotJSc3)

	// verify job update
	job1.Status = models.JobStatusSuccessful
	job1.Result = &models.JobResult{
		StdOut: "some std out",
		StdErr: "some std err",
	}
	ft := time.Date(2020, 11, 5, 12, 11, 20, 0, time.UTC)
	job1.FinishedAt = &ft

	require.NoError(t, p.SaveJob(job1))
	gotJob1, err = p.GetByJID(job1.ClientID, job1.JID)
	require.NoError(t, err)
	require.NotNil(t, gotJob1)
	assert.Equal(t, job1, gotJob1)

	gotJSc1, err = p.List(ctx, &query.ListOptions{Filters: []query.FilterOption{{Column: []string{"client_id"}, Values: []string{job1.ClientID}}}})
	require.NoError(t, err)
	assert.ElementsMatch(t, []*models.Job{job1, job2}, gotJSc1)
}

func TestCreateJob(t *testing.T) {
	jobsDB, err := sqlite.New(":memory:", jobs.AssetNames(), jobs.Asset, DataSourceOptions)
	require.NoError(t, err)
	p := NewSqliteProvider(jobsDB, testLog)
	defer p.Close()

	// create job
	job := jb.New(t).Status(models.JobStatusSuccessful).Result(nil).Build()
	require.NoError(t, p.CreateJob(job))

	// try to create the same job but with different status
	updatedJob := *job
	updatedJob.Status = models.JobStatusRunning
	require.NoError(t, p.CreateJob(job))

	// verify the job contains the initial status
	gotJob, err := p.GetByJID(job.ClientID, job.JID)
	require.NoError(t, err)
	require.Equal(t, job, gotJob)
}
