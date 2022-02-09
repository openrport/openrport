package jobs

import (
	"context"

	"github.com/pkg/errors"
)

type CleanupProvider interface {
	CleanupJobsMultiJobs(context.Context, int) error
}

type CleanupTask struct {
	provider CleanupProvider
	maxJobs  int
}

func NewCleanupTask(provider CleanupProvider, maxJobs int) *CleanupTask {
	return &CleanupTask{
		provider: provider,
		maxJobs:  maxJobs,
	}
}

func (t *CleanupTask) Run(ctx context.Context) error {
	return t.provider.CleanupJobsMultiJobs(ctx, t.maxJobs)
}

func (p *SqliteProvider) CleanupJobsMultiJobs(ctx context.Context, maxJobs int) error {
	// Delete all multi jobs that have jobs after max jobs
	_, err := p.db.ExecContext(ctx, "DELETE FROM multi_jobs WHERE jid IN (SELECT multi_job_id FROM jobs ORDER BY started_at DESC LIMIT -1 OFFSET ?)", maxJobs)
	if err != nil {
		return errors.Wrap(err, "deleting multi jobs")
	}
	// Delete all jobs associated with multi jobs
	_, err = p.db.ExecContext(ctx, "DELETE FROM jobs WHERE multi_job_id IN (SELECT multi_job_id FROM jobs ORDER BY started_at DESC LIMIT -1 OFFSET ?) AND multi_job_id IS NOT NULL", maxJobs)
	if err != nil {
		return errors.Wrap(err, "deleting multi jobs' jobs")
	}
	// Delete any jobs left not from multi jobs
	_, err = p.db.ExecContext(ctx, "DELETE FROM jobs WHERE jid IN (SELECT jid FROM jobs ORDER BY started_at DESC LIMIT -1 OFFSET ?) AND multi_job_id IS NULL", maxJobs)
	if err != nil {
		return errors.Wrap(err, "deleting jobs")
	}

	return nil
}
