package jobs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"

	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

const (
	DefaultLimit = 100
	MaxLimit     = 1000
)

var JobSupportedFilters = map[string]bool{
	"jid":                true,
	"started_at[gt]":     true,
	"started_at[lt]":     true,
	"started_at[since]":  true,
	"started_at[until]":  true,
	"finished_at[gt]":    true,
	"finished_at[lt]":    true,
	"finished_at[since]": true,
	"finished_at[until]": true,
	"status":             true,
	"created_by":         true,
	"multi_job_id":       true,
	"schedule_id":        true,
	"client_id":          true,
}
var JobSupportedSorts = map[string]bool{
	"jid":          true,
	"started_at":   true,
	"finished_at":  true,
	"status":       true,
	"multi_job_id": true,
	"schedule_id":  true,
	"created_by":   true,
}
var jobFields = map[string]bool{
	"jid":          true,
	"status":       true,
	"finished_at":  true,
	"client_id":    true,
	"client_name":  true,
	"command":      true,
	"cwd":          true,
	"interpreter":  true,
	"pid":          true,
	"started_at":   true,
	"created_by":   true,
	"timeout_sec":  true,
	"multi_job_id": true,
	"schedule_id":  true,
	"error":        true,
	"is_sudo":      true,
	"is_script":    true,
}
var JobSupportedFields = map[string]map[string]bool{
	"jobs":     jobFields,
	"commands": jobFields,
	"scripts":  jobFields,
	"result": {
		"stdout":  true,
		"stderr":  true,
		"summary": true,
	},
}
var JobListDefaultFields = map[string][]string{
	"fields[jobs]": {
		"jid",
		"status",
		"finished_at",
	},
	"fields[result]": {
		"summary",
	},
}

type SqliteProvider struct {
	log *logger.Logger
	db  *sqlx.DB
}

func NewSqliteProvider(db *sqlx.DB, log *logger.Logger) *SqliteProvider {
	return &SqliteProvider{db: db, log: log}
}

func (p *SqliteProvider) GetByJID(clientID, jid string) (*models.Job, error) {
	res := &jobSqlite{}
	err := p.db.Get(res, "SELECT jobs.*, schedule_id FROM jobs LEFT JOIN multi_jobs ON jobs.multi_job_id = multi_jobs.jid WHERE jobs.jid=?", jid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return res.convert(), nil
}

func (p *SqliteProvider) List(ctx context.Context, options *query.ListOptions) ([]*models.Job, error) {
	if len(options.Sorts) == 0 {
		options.Sorts = []query.SortOption{
			{
				Column: "finished_at",
				IsASC:  false,
			},
			{
				Column: "jid",
				IsASC:  true,
			},
		}
	}

	q := "SELECT jobs.*, schedule_id FROM jobs LEFT JOIN multi_jobs ON jobs.multi_job_id = multi_jobs.jid"
	q, params := query.AppendOptionsToQuery(options, q, nil)

	var res []*jobSqlite
	err := p.db.SelectContext(ctx, &res, q, params...)
	if err != nil {
		return nil, err
	}
	return convertJobs(res), nil
}

func (p *SqliteProvider) Count(ctx context.Context, options *query.ListOptions) (int, error) {
	countOptions := *options
	countOptions.Pagination = nil

	q := "SELECT count(*) FROM (SELECT jobs.*, schedule_id FROM jobs LEFT JOIN multi_jobs ON jobs.multi_job_id = multi_jobs.jid)"
	q, params := query.AppendOptionsToQuery(&countOptions, q, nil)

	var result int
	err := p.db.GetContext(ctx, &result, q, params...)
	if err != nil {
		return 0, err
	}
	return result, nil
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
	JID        string         `db:"jid"`
	Status     string         `db:"status"`
	FinishedAt sql.NullTime   `db:"finished_at"`
	StartedAt  time.Time      `db:"started_at"`
	CreatedBy  string         `db:"created_by"`
	ClientID   string         `db:"client_id"`
	MultiJobID sql.NullString `db:"multi_job_id"`
	ScheduleID *string        `db:"schedule_id"`
	Details    *jobDetails    `db:"details"`
}

type jobDetails struct {
	Command     string            `json:"command"`
	Cwd         string            `json:"cwd"`
	IsSudo      bool              `json:"is_sudo"`
	IsScript    bool              `json:"is_script"`
	Interpreter string            `json:"interpreter"`
	PID         *int              `json:"pid"`
	TimeoutSec  int               `json:"timeout_sec"`
	Error       string            `json:"error"`
	Result      *models.JobResult `json:"result"`
	ClientName  string            `json:"client_name"`
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

func convertJobs(list []*jobSqlite) []*models.Job {
	res := make([]*models.Job, 0, len(list))
	for _, cur := range list {
		res = append(res, cur.convert())
	}
	return res
}

func (j *jobSqlite) convert() *models.Job {
	res := &models.Job{
		JID:        j.JID,
		Status:     j.Status,
		ClientID:   j.ClientID,
		StartedAt:  j.StartedAt,
		CreatedBy:  j.CreatedBy,
		ScheduleID: j.ScheduleID,
	}
	if j.Details != nil {
		res.ClientName = j.Details.ClientName
		res.Command = j.Details.Command
		res.Interpreter = j.Details.Interpreter
		res.PID = j.Details.PID
		res.TimeoutSec = j.Details.TimeoutSec
		res.Result = j.Details.Result
		res.Error = j.Details.Error
		res.Cwd = j.Details.Cwd
		res.IsSudo = j.Details.IsSudo
		res.IsScript = j.Details.IsScript
	}
	if j.FinishedAt.Valid {
		res.FinishedAt = &j.FinishedAt.Time
	}
	if j.MultiJobID.Valid {
		res.MultiJobID = &j.MultiJobID.String
	}
	return res
}

func convertToSqlite(job *models.Job) *jobSqlite {
	res := &jobSqlite{
		JID:       job.JID,
		Status:    job.Status,
		StartedAt: job.StartedAt,
		CreatedBy: job.CreatedBy,
		ClientID:  job.ClientID,
		Details: &jobDetails{
			Command:     job.Command,
			Interpreter: job.Interpreter,
			PID:         job.PID,
			TimeoutSec:  job.TimeoutSec,
			Result:      job.Result,
			Error:       job.Error,
			ClientName:  job.ClientName,
			Cwd:         job.Cwd,
			IsSudo:      job.IsSudo,
			IsScript:    job.IsScript,
		},
	}
	if job.MultiJobID != nil {
		res.MultiJobID = sql.NullString{String: *job.MultiJobID, Valid: true}
	}
	if job.FinishedAt != nil {
		res.FinishedAt = sql.NullTime{Time: *job.FinishedAt, Valid: true}
	}
	return res
}
