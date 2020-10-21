package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/server/test/jb"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func TestSortByFinishedAt(t *testing.T) {
	// given
	ft := time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC)
	j1 := jb.New(t).JID("1").Status(models.JobStatusRunning).Build().JobSummary
	j2 := jb.New(t).JID("2").Status(models.JobStatusRunning).Build().JobSummary
	j3 := jb.New(t).JID("3").Status(models.JobStatusRunning).Build().JobSummary
	j4 := jb.New(t).JID("4").Status(models.JobStatusFailed).FinishedAt(ft.Add(time.Minute)).Build().JobSummary
	j5 := jb.New(t).JID("5").Status(models.JobStatusSuccessful).FinishedAt(ft).Build().JobSummary
	j6 := jb.New(t).JID("6").Status(models.JobStatusUnknown).FinishedAt(ft.Add(-time.Hour)).Build().JobSummary

	testCases := []struct {
		name     string
		desc     bool
		jobs     []*models.JobSummary
		wantJobs []*models.JobSummary
	}{
		{
			name:     "desc",
			desc:     true,
			jobs:     []*models.JobSummary{&j2, &j6, &j1, &j4, &j3, &j5},
			wantJobs: []*models.JobSummary{&j1, &j2, &j3, &j4, &j5, &j6},
		},
		{
			name:     "asc",
			desc:     false,
			jobs:     []*models.JobSummary{&j2, &j6, &j1, &j4, &j3, &j5},
			wantJobs: []*models.JobSummary{&j6, &j5, &j4, &j3, &j2, &j1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			SortByFinishedAt(tc.jobs, tc.desc)

			// then
			assert.Equal(t, tc.wantJobs, tc.jobs)
		})
	}
}
