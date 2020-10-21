package models

import (
	"time"
)

const (
	JobStatusSuccessful = "successful"
	JobStatusRunning    = "running"
	JobStatusFailed     = "failed"
	JobStatusUnknown    = "unknown"
)

type Job struct {
	JobSummary
	SID       string        `json:"sid"`
	Command   string        `json:"command"`
	PID       int           `json:"pid"`
	StartedAt time.Time     `json:"started_at"`
	CreatedBy string        `json:"created_by"`
	Timeout   time.Duration `json:"timeout"`
	Result    *JobResult    `json:"result"`
}

// JobSummary short info about a job.
type JobSummary struct {
	JID        string     `json:"jid"`
	Status     string     `json:"status"`
	FinishedAt *time.Time `json:"finished_at"`
}

type JobResult struct {
	StdOut string `json:"stdout"`
	StdErr string `json:"stderr"`
}
