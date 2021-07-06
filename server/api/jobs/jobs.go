package jobs

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"

	"github.com/cloudradar-monitoring/rport/db/migration/jobs"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type SqliteProvider struct {
	log *chshare.Logger
	db  *sqlx.DB
}

func NewSqliteProvider(dbPath string, log *chshare.Logger) (*SqliteProvider, error) {
	db, err := sqlite.New(dbPath, jobs.AssetNames(), jobs.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to create jobs DB instance: %v", err)
	}
	return &SqliteProvider{db: db, log: log}, nil
}

func (p *SqliteProvider) GetByJID(clientID, jid string) (*models.Job, error) {
	res := &jobSqlite{}
	err := p.db.Get(res, "SELECT * FROM jobs WHERE jid=?", jid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return res.convert(), nil
}

// GetByMultiJobID returns a list of all jobs that belongs to a multi-client job with a given ID sorted by started_at(desc), jid order.
func (p *SqliteProvider) GetByMultiJobID(jid string) ([]*models.Job, error) {
	var res []*jobSqlite
	err := p.db.Select(&res, "SELECT * FROM jobs WHERE multi_job_id=? ORDER BY DATETIME(started_at) DESC, jid", jid)
	if err != nil {
		return nil, err
	}
	return convertJobs(res), nil
}

func (p *SqliteProvider) GetSummariesByClientID(clientID string) ([]*models.JobSummary, error) {
	var res []*jobSummarySqlite
	err := p.db.Select(&res, "SELECT jid, finished_at, status FROM jobs WHERE client_id=?", clientID)
	if err != nil {
		return nil, err
	}
	return convertJSs(res), nil
}

// SaveJob creates a new or updates an existing job.
func (p *SqliteProvider) SaveJob(job *models.Job) error {
	_, err := p.db.NamedExec(`INSERT OR REPLACE INTO jobs (jid, status, started_at, finished_at, created_by, client_id, multi_job_id, details)
														VALUES (:jid, :status, :started_at, :finished_at, :created_by, :client_id, :multi_job_id, :details)`,
		convertToSqlite(job))
	if err == nil {
		p.log.Debugf("Job saved successfully: %v", *job)
	}
	return err
}

// CreateJob creates a new job. If already exists with the same ID - does nothing and returns nil.
func (p *SqliteProvider) CreateJob(job *models.Job) error {
	_, err := p.db.NamedExec(`INSERT INTO jobs (jid, status, started_at, finished_at, created_by, client_id, multi_job_id, details)
											VALUES (:jid, :status, :started_at, :finished_at, :created_by, :client_id, :multi_job_id, :details)`,
		convertToSqlite(job))
	if err != nil {
		// check if it's "already exist" err
		typeErr, ok := err.(sqlite3.Error)
		if ok && typeErr.Code == sqlite3.ErrConstraint {
			p.log.Debugf("Job already exist with ID: %s", job.JID)
			return nil
		}
	} else {
		p.log.Debugf("Job saved successfully: %v", *job)
	}
	return err
}

func (p *SqliteProvider) Close() error {
	return p.db.Close()
}

type jobSqlite struct {
	jobSummarySqlite
	StartedAt  time.Time      `db:"started_at"`
	CreatedBy  string         `db:"created_by"`
	ClientID   string         `db:"client_id"`
	MultiJobID sql.NullString `db:"multi_job_id"`
	Details    *jobDetails    `db:"details"`
}

type jobSummarySqlite struct {
	JID        string       `db:"jid"`
	Status     string       `db:"status"`
	FinishedAt sql.NullTime `db:"finished_at"`
}

type jobDetails struct {
	Command    string            `json:"command"`
	Cwd        string            `json:"cwd"`
	IsSudo     bool              `json:"sudo"`
	IsScript   bool              `json:"is_script"`
	Shell      string            `json:"shell"`
	PID        *int              `json:"pid"`
	TimeoutSec int               `json:"timeout_sec"`
	Error      string            `json:"error"`
	Result     *models.JobResult `json:"result"`
	ClientName string            `json:"client_name"`
}

func (d *jobDetails) Scan(value interface{}) error {
	if d == nil {
		return errors.New("'details' cannot be nil")
	}
	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected to have string, got %T", value)
	}
	err := json.Unmarshal([]byte(valueStr), d)
	if err != nil {
		return fmt.Errorf("failed to decode 'details' field: %v", err)
	}
	return nil
}

func (d *jobDetails) Value() (driver.Value, error) {
	if d == nil {
		return nil, errors.New("'details' cannot be nil")
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("failed to encode 'details' field: %v", err)
	}
	return string(b), nil
}

func (js *jobSummarySqlite) convert() *models.JobSummary {
	res := &models.JobSummary{
		JID:    js.JID,
		Status: js.Status,
	}
	if js.FinishedAt.Valid {
		res.FinishedAt = &js.FinishedAt.Time
	}
	return res
}

func convertJSs(list []*jobSummarySqlite) []*models.JobSummary {
	res := make([]*models.JobSummary, 0, len(list))
	for _, cur := range list {
		res = append(res, cur.convert())
	}
	return res
}

func (j *jobSqlite) convert() *models.Job {
	js := j.jobSummarySqlite.convert()
	res := &models.Job{
		JobSummary: *js,
		ClientID:   j.ClientID,
		ClientName: j.Details.ClientName,
		StartedAt:  j.StartedAt,
		CreatedBy:  j.CreatedBy,
		Command:    j.Details.Command,
		Shell:      j.Details.Shell,
		PID:        j.Details.PID,
		TimeoutSec: j.Details.TimeoutSec,
		Result:     j.Details.Result,
		Error:      j.Details.Error,
		Cwd:        j.Details.Cwd,
		IsSudo:     j.Details.IsSudo,
		IsScript:   j.Details.IsScript,
	}
	if j.MultiJobID.Valid {
		res.MultiJobID = &j.MultiJobID.String
	}
	return res
}

func convertJobs(list []*jobSqlite) []*models.Job {
	res := make([]*models.Job, 0, len(list))
	for _, cur := range list {
		res = append(res, cur.convert())
	}
	return res
}

func convertToSqlite(job *models.Job) *jobSqlite {
	res := &jobSqlite{
		jobSummarySqlite: jobSummarySqlite{
			JID:    job.JID,
			Status: job.Status,
		},
		StartedAt: job.StartedAt,
		CreatedBy: job.CreatedBy,
		ClientID:  job.ClientID,
		Details: &jobDetails{
			Command:    job.Command,
			Shell:      job.Shell,
			PID:        job.PID,
			TimeoutSec: job.TimeoutSec,
			Result:     job.Result,
			Error:      job.Error,
			ClientName: job.ClientName,
			Cwd:        job.Cwd,
			IsSudo:     job.IsSudo,
			IsScript:   job.IsScript,
		},
	}
	if job.MultiJobID != nil {
		res.MultiJobID = sql.NullString{String: *job.MultiJobID, Valid: true}
	}
	if job.FinishedAt != nil {
		res.jobSummarySqlite.FinishedAt = sql.NullTime{Time: *job.FinishedAt, Valid: true}
	}
	return res
}
