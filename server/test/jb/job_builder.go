// Generating data for tests is always cumbersome.
// To make it easier this package should be a single source of truth for generating Jobs data.
//
// This package provides a builder that can generate Jobs with:
// - preset fields,
// - randomly generated fields,
// - fields set on demand.
//
// It can be extended by needs.
package jb

import (
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
)

type JobBuilder struct {
	t *testing.T

	jid        string
	clientID   string
	clientName string
	multiJobID string
	status     string
	startedAt  time.Time
	finishedAt *time.Time
	result     *models.JobResult
}

// New returns a builder to generate a job that can be used in tests.
func New(t *testing.T) JobBuilder {
	return JobBuilder{
		t:          t,
		clientID:   generateRandomCID(),
		clientName: generateRandomClientName(),
		status:     models.JobStatusSuccessful,
		startedAt:  time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC),
		result: &models.JobResult{
			StdOut: "Mon Sep 28 09:05:08 UTC 2020\nrport",
			StdErr: "/bin/sh: 1: foo: not found",
		},
	}
}

func (b JobBuilder) JID(jid string) JobBuilder {
	b.jid = jid
	return b
}

func (b JobBuilder) ClientID(clientID string) JobBuilder {
	b.clientID = clientID
	return b
}

func (b JobBuilder) ClientName(clientName string) JobBuilder {
	b.clientName = clientName
	return b
}

func (b JobBuilder) MultiJobID(multiJobID string) JobBuilder {
	b.multiJobID = multiJobID
	return b
}

func (b JobBuilder) Status(status string) JobBuilder {
	b.status = status
	return b
}

func (b JobBuilder) StartedAt(startedAt time.Time) JobBuilder {
	b.startedAt = startedAt
	return b
}

func (b JobBuilder) FinishedAt(finishedAt time.Time) JobBuilder {
	b.finishedAt = &finishedAt
	return b
}

func (b JobBuilder) Result(result *models.JobResult) JobBuilder {
	b.result = result
	return b
}

func (b JobBuilder) Build() *models.Job {
	if b.jid == "" {
		b.jid = generateRandomJID()
	}
	pid := 1245
	// hardcoded values are used because currently was no need of other data, extend with more available options if needed
	return &models.Job{
		JobSummary: models.JobSummary{
			JID:        b.jid,
			Status:     b.status,
			FinishedAt: b.finishedAt,
		},
		ClientID:   b.clientID,
		ClientName: b.clientName,
		Command:    "/bin/date;foo;whoami",
		PID:        &pid,
		StartedAt:  b.startedAt,
		CreatedBy:  "test-user",
		TimeoutSec: 60,
		Result:     b.result,
		MultiJobID: &b.multiJobID,
	}
}

func generateRandomCID() string {
	return "cid-" + random.AlphaNum(12)
}

func generateRandomClientName() string {
	return "client-name-" + random.AlphaNum(12)
}

func generateRandomJID() string {
	return "jid-" + random.UUID4()
}
