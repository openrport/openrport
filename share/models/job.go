package models

import (
	"fmt"
	"strings"
	"time"
)

const (
	JobStatusSuccessful = "successful"
	JobStatusRunning    = "running"
	JobStatusFailed     = "failed"
	JobStatusUnknown    = "unknown"

	ChannelStdout = "stdout"
	ChannelStderr = "stderr"
)

type Job struct {
	JID          string     `json:"jid"`
	Status       string     `json:"status"`
	FinishedAt   *time.Time `json:"finished_at"`
	ClientID     string     `json:"client_id"`
	ClientName   string     `json:"client_name"`
	Command      string     `json:"command"`
	Cwd          string     `json:"cwd"`
	Interpreter  string     `json:"interpreter"`
	PID          *int       `json:"pid"`
	StartedAt    time.Time  `json:"started_at"`
	CreatedBy    string     `json:"created_by"`
	TimeoutSec   int        `json:"timeout_sec"`
	MultiJobID   *string    `json:"multi_job_id"`
	ScheduleID   *string    `json:"schedule_id"`
	Error        string     `json:"error"`
	Result       *JobResult `json:"result"`
	IsSudo       bool       `json:"is_sudo"`
	IsScript     bool       `json:"is_script"`
	StreamResult bool       `json:"stream_result"`
}

type JobResult struct {
	StdOut  string `json:"stdout"`
	StdErr  string `json:"stderr"`
	Summary string `json:"summary"`
}

type JobClientTags struct {
	Tags     []string `json:"tags"`
	Operator string   `json:"operator"`
}

// TODO: check that ClientTags is populated where required
type MultiJob struct {
	MultiJobSummary
	ClientIDs   []string       `json:"client_ids"`
	GroupIDs    []string       `json:"group_ids"`
	ClientTags  *JobClientTags `json:"tags"`
	Command     string         `json:"command"`
	Cwd         string         `json:"cwd"`
	Interpreter string         `json:"interpreter"`
	TimeoutSec  int            `json:"timeout_sec"`
	Concurrent  bool           `json:"concurrent"`
	AbortOnErr  bool           `json:"abort_on_err"`
	Jobs        []*Job         `json:"jobs"`
	IsSudo      bool           `json:"is_sudo"`
	IsScript    bool           `json:"is_script"`
}

type MultiJobSummary struct {
	JID        string    `json:"jid"`
	StartedAt  time.Time `json:"started_at"`
	CreatedBy  string    `json:"created_by"`
	ScheduleID *string   `json:"schedule_id"`
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

// TODO: add some unit tests. not high priority but good to get done.
func (jct *JobClientTags) String() string {
	var str string
	if jct == nil {
		return "[]"
	}
	numTags := len(jct.Tags)
	if numTags == 0 {
		return "[]"
	}
	tagsList := strings.Join(jct.Tags, ",")
	if numTags == 1 {
		str = fmt.Sprintf("[%s]", tagsList)
	} else {
		operator := jct.Operator
		if operator == "" {
			operator = "OR"
		}
		str = fmt.Sprintf("[%s: %s]", operator, tagsList)
	}
	return str
}
