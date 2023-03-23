package jb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/share/models"
)

type MultiJobBuilder struct {
	t *testing.T

	jid        string
	scheduleID *string
	clientIDs  []string
	startedAt  time.Time
	concurrent bool
	abortOnErr bool
	withJobs   bool
	sudo       bool
	cwd        string
}

// NewMulti returns a builder to generate a multi-client job that can be used in tests.
func NewMulti(t *testing.T) MultiJobBuilder {
	return MultiJobBuilder{
		t:         t,
		startedAt: time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC),
	}
}

func (b MultiJobBuilder) JID(jid string) MultiJobBuilder {
	b.jid = jid
	return b
}

func (b MultiJobBuilder) ScheduleID(scheduleID string) MultiJobBuilder {
	b.scheduleID = &scheduleID
	return b
}

func (b MultiJobBuilder) ClientIDs(clientIDs ...string) MultiJobBuilder {
	b.clientIDs = append(b.clientIDs, clientIDs...)
	return b
}

func (b MultiJobBuilder) WithJobs() MultiJobBuilder {
	b.withJobs = true
	return b
}

func (b MultiJobBuilder) Concurrent(concurrent bool) MultiJobBuilder {
	b.concurrent = concurrent
	return b
}

func (b MultiJobBuilder) WithSudo() MultiJobBuilder {
	b.sudo = true
	return b
}

func (b MultiJobBuilder) WithCwd(cwd string) MultiJobBuilder {
	b.cwd = cwd
	return b
}

func (b MultiJobBuilder) AbortOnErr(abortOnErr bool) MultiJobBuilder {
	b.abortOnErr = abortOnErr
	return b
}

func (b MultiJobBuilder) StartedAt(startedAt time.Time) MultiJobBuilder {
	b.startedAt = startedAt
	return b
}

func (b MultiJobBuilder) Build() *models.MultiJob {
	if b.jid == "" {
		jid, err := generateRandomJID()
		require.NoError(b.t, err)
		b.jid = jid
	}
	if len(b.clientIDs) == 0 {
		b.clientIDs = []string{generateRandomCID(), generateRandomCID()}
	}
	jobs := []*models.Job{}
	if b.withJobs {
		st := b.startedAt.Add(time.Minute) // is used to order jobs to make tests work
		for _, clientID := range b.clientIDs {
			j := New(b.t).ClientID(clientID).MultiJobID(b.jid).StartedAt(st).Build()
			jobs = append(jobs, j)
			st = st.Add(-time.Second)
		}
	}
	return &models.MultiJob{
		MultiJobSummary: models.MultiJobSummary{
			JID:        b.jid,
			StartedAt:  b.startedAt,
			CreatedBy:  "test-user",
			ScheduleID: b.scheduleID,
		},
		ClientIDs:  b.clientIDs,
		Command:    "/bin/date;foo;whoami",
		Cwd:        b.cwd,
		IsSudo:     b.sudo,
		TimeoutSec: 60,
		Concurrent: b.concurrent,
		AbortOnErr: b.abortOnErr,
		Jobs:       jobs,
	}
}
