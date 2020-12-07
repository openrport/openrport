package jobs

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/migration/jobs"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type SqliteProvider struct {
	db *sqlx.DB
}

func NewSqliteProvider(dbPath string) (*SqliteProvider, error) {
	db, err := sqlite.New(dbPath, jobs.AssetNames(), jobs.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to create jobs DB instance: %v", err)
	}
	return &SqliteProvider{db: db}, nil
}

func (p *SqliteProvider) GetByJID(sid, jid string) (*models.Job, error) {
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

func (p *SqliteProvider) GetSummariesBySID(sid string) ([]*models.JobSummary, error) {
	var res []*jobSummarySqlite
	err := p.db.Select(&res, "SELECT jid, finished_at, status FROM jobs WHERE sid=?", sid)
	if err != nil {
		return nil, err
	}
	return convertJSs(res), nil
}

func (p *SqliteProvider) SaveJob(job *models.Job) error {
	_, err := p.db.NamedExec(`INSERT OR REPLACE INTO jobs (jid, status, started_at, finished_at, created_by, sid, details)
														VALUES (:jid, :status, :started_at, :finished_at, :created_by, :sid, :details)`,
		convertToSqlite(job))
	return err
}

func (p *SqliteProvider) Close() error {
	return p.db.Close()
}

type jobSqlite struct {
	jobSummarySqlite
	StartedAt time.Time   `db:"started_at"`
	CreatedBy string      `db:"created_by"`
	SID       string      `db:"sid"`
	Details   *jobDetails `db:"details"`
}

type jobSummarySqlite struct {
	JID        string       `db:"jid"`
	Status     string       `db:"status"`
	FinishedAt sql.NullTime `db:"finished_at"`
}

type jobDetails struct {
	Command    string            `json:"command"`
	Shell      string            `json:"shell"`
	PID        int               `json:"pid"`
	TimeoutSec int               `json:"timeout_sec"`
	Result     *models.JobResult `json:"result"`
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
	return &models.Job{
		JobSummary: *js,
		SID:        j.SID,
		StartedAt:  j.StartedAt,
		CreatedBy:  j.CreatedBy,
		Command:    j.Details.Command,
		Shell:      j.Details.Shell,
		PID:        j.Details.PID,
		TimeoutSec: j.Details.TimeoutSec,
		Result:     j.Details.Result,
	}
}

func convertToSqlite(job *models.Job) *jobSqlite {
	res := &jobSqlite{
		jobSummarySqlite: jobSummarySqlite{
			JID:    job.JID,
			Status: job.Status,
		},
		StartedAt: job.StartedAt,
		CreatedBy: job.CreatedBy,
		SID:       job.SID,
		Details: &jobDetails{
			Command:    job.Command,
			Shell:      job.Shell,
			PID:        job.PID,
			TimeoutSec: job.TimeoutSec,
			Result:     job.Result,
		},
	}
	if job.FinishedAt != nil {
		res.jobSummarySqlite.FinishedAt = sql.NullTime{Time: *job.FinishedAt, Valid: true}
	}
	return res
}
