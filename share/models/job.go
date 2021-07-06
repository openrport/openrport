package models

import (
	"fmt"
	"os"
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
	ClientID   string     `json:"client_id"`
	ClientName string     `json:"client_name"`
	Command    string     `json:"command"`
	Cwd        string     `json:"cwd"`
	Shell      string     `json:"shell"`
	PID        *int       `json:"pid"`
	StartedAt  time.Time  `json:"started_at"`
	CreatedBy  string     `json:"created_by"`
	TimeoutSec int        `json:"timeout_sec"`
	MultiJobID *string    `json:"multi_job_id"`
	Error      string     `json:"error"`
	Result     *JobResult `json:"result"`
	IsSudo     bool       `json:"sudo"`
	IsScript   bool       `json:"is_script"`
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

type MultiJob struct {
	MultiJobSummary
	ClientIDs  []string `json:"client_ids"`
	GroupIDs   []string `json:"group_ids"`
	Command    string   `json:"command"`
	Cwd        string   `json:"cwd"`
	Shell      string   `json:"shell"`
	TimeoutSec int      `json:"timeout_sec"`
	Concurrent bool     `json:"concurrent"`
	AbortOnErr bool     `json:"abort_on_err"`
	Jobs       []*Job   `json:"jobs"`
	IsSudo     bool     `json:"sudo"`
	IsScript   bool     `json:"is_script"`
}

type MultiJobSummary struct {
	JID       string    `json:"jid"`
	StartedAt time.Time `json:"started_at"`
	CreatedBy string    `json:"created_by"`
}

type MultiJobResult struct {
	Status string     `json:"status"`
	StdErr string     `json:"stderr"`
	Result *JobResult `json:"result"`
}

func (j Job) LogPrefix() string {
	var r string
	if j.MultiJobID != nil {
		r = fmt.Sprintf("multiJobID=%q, ", *j.MultiJobID)
	}
	r += fmt.Sprintf("jid=%q, clientID=%q", j.JID, j.ClientID)
	return r
}

type File struct {
	Name      string      `json:"name"`
	Content   []byte      `json:"content"`
	CreateDir bool        `json:"create_dir"`
	Mode      os.FileMode `json:"file_mode"`
}
