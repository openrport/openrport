package jobs

import (
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/test/jb"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func TestJobsSqliteProvider(t *testing.T) {
	p, err := NewSqliteProvider(":memory:")
	require.NoError(t, err)
	defer p.Close()

	// add jobs
	job1 := jb.New(t).Status(models.JobStatusRunning).Result(nil).Build()
	job2 := jb.New(t).SID(job1.SID).Build()
	job3 := jb.New(t).Build() // different sid
	require.NoError(t, p.SaveJob(job1))
	require.NoError(t, p.SaveJob(job2))
	require.NoError(t, p.SaveJob(job3))

	// verify added jobs
	gotJob1, err := p.GetByJID(job1.SID, job1.JID)
	require.NoError(t, err)
	require.NotNil(t, gotJob1)
	assert.Equal(t, job1, gotJob1)

	gotJob2, err := p.GetByJID(job2.SID, job2.JID)
	require.NoError(t, err)
	require.NotNil(t, gotJob2)
	assert.Equal(t, job2, gotJob2)

	gotJob3, err := p.GetByJID(job3.SID, job3.JID)
	require.NoError(t, err)
	require.NotNil(t, gotJob3)
	assert.Equal(t, job3, gotJob3)

	// verify not found job
	gotJob4, err := p.GetByJID(job3.SID, "unknown-jid")
	require.NoError(t, err)
	require.Nil(t, gotJob4)

	// verify job summaries
	gotJSs1, err := p.GetSummariesBySID(job1.SID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*models.JobSummary{&job1.JobSummary, &job2.JobSummary}, gotJSs1)

	gotJSs2, err := p.GetSummariesBySID(job3.SID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*models.JobSummary{&job3.JobSummary}, gotJSs2)

	// verify job summaries not found
	gotJSs3, err := p.GetSummariesBySID("unknown-sid")
	require.NoError(t, err)
	require.Empty(t, gotJSs3)

	// verify job update
	job1.Status = models.JobStatusSuccessful
	job1.Result = &models.JobResult{
		StdOut: "some std out",
		StdErr: "some std err",
	}
	ft := time.Date(2020, 11, 5, 12, 11, 20, 0, time.UTC)
	job1.FinishedAt = &ft

	require.NoError(t, p.SaveJob(job1))
	gotJob1, err = p.GetByJID(job1.SID, job1.JID)
	require.NoError(t, err)
	require.NotNil(t, gotJob1)
	assert.Equal(t, job1, gotJob1)

	gotJSs1, err = p.GetSummariesBySID(job1.SID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*models.JobSummary{&job1.JobSummary, &job2.JobSummary}, gotJSs1)
}
