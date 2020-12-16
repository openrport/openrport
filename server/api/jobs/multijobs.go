package jobs

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cloudradar-monitoring/rport/share/models"
)

// GetMultiJob returns a multi-client job with fetched all clients' jobs.
func (p *SqliteProvider) GetMultiJob(jid string) (*models.MultiJob, error) {
	res := &multiJobSqlite{}
	err := p.db.Get(res, "SELECT * FROM multi_jobs WHERE jid=?", jid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	multiJob := res.convert()

	// fetch clients' jobs
	jobs, err := p.GetByMultiJobID(jid)
	if err != nil {
		return nil, err
	}
	multiJob.Jobs = jobs

	return multiJob, nil
}

// GetAllMultiJobSummaries returns a list of summaries of all multi-clients jobs sorted by started_at(desc), jid order.
func (p *SqliteProvider) GetAllMultiJobSummaries() ([]*models.MultiJobSummary, error) {
	var res []*multiJobSummarySqlite
	err := p.db.Select(&res, "SELECT jid, started_at, created_by FROM multi_jobs ORDER BY DATETIME(started_at) DESC, jid")
	if err != nil {
		return nil, err
	}
	return convertMultiJSs(res), nil
}

// SaveMultiJob creates a new or updates an existing multi-client job (without child jobs).
func (p *SqliteProvider) SaveMultiJob(job *models.MultiJob) error {
	_, err := p.db.NamedExec(`INSERT OR REPLACE INTO multi_jobs (jid, started_at, created_by, details)
															  VALUES (:jid, :started_at, :created_by, :details)`,
		convertMultiJobToSqlite(job))
	if err == nil {
		p.log.Debugf("Multi-client Job saved successfully: %v", *job)
	}
	return err
}

type multiJobSqlite struct {
	multiJobSummarySqlite
	Details *multiJobDetailSqlite `db:"details"`
}

type multiJobSummarySqlite struct {
	JID       string    `db:"jid"`
	StartedAt time.Time `db:"started_at"`
	CreatedBy string    `db:"created_by"`
}

type multiJobDetailSqlite struct {
	ClientIDs  []string `json:"client_ids"`
	Command    string   `json:"command"`
	Shell      string   `json:"shell"`
	TimeoutSec int      `json:"timeout_sec"`
	Concurrent bool     `json:"concurrent"`
	AbortOnErr bool     `json:"abort_on_err"`
}

func (d *multiJobDetailSqlite) Scan(value interface{}) error {
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

func (d *multiJobDetailSqlite) Value() (driver.Value, error) {
	if d == nil {
		return nil, errors.New("'details' cannot be nil")
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("failed to encode 'details' field: %v", err)
	}
	return string(b), nil
}

func (js *multiJobSummarySqlite) convert() *models.MultiJobSummary {
	return &models.MultiJobSummary{
		JID:       js.JID,
		StartedAt: js.StartedAt,
		CreatedBy: js.CreatedBy,
	}
}

func convertMultiJSs(list []*multiJobSummarySqlite) []*models.MultiJobSummary {
	res := make([]*models.MultiJobSummary, 0, len(list))
	for _, cur := range list {
		res = append(res, cur.convert())
	}
	return res
}

func (j *multiJobSqlite) convert() *models.MultiJob {
	js := j.multiJobSummarySqlite.convert()
	d := j.Details
	return &models.MultiJob{
		MultiJobSummary: *js,
		ClientIDs:       d.ClientIDs,
		Command:         d.Command,
		Shell:           d.Shell,
		TimeoutSec:      d.TimeoutSec,
		Concurrent:      d.Concurrent,
		AbortOnErr:      d.AbortOnErr,
	}
}

func convertMultiJobToSqlite(job *models.MultiJob) *multiJobSqlite {
	return &multiJobSqlite{
		multiJobSummarySqlite: multiJobSummarySqlite{
			JID:       job.JID,
			StartedAt: job.StartedAt,
			CreatedBy: job.CreatedBy,
		},
		Details: &multiJobDetailSqlite{
			ClientIDs:  job.ClientIDs,
			Command:    job.Command,
			Shell:      job.Shell,
			TimeoutSec: job.TimeoutSec,
			Concurrent: job.Concurrent,
			AbortOnErr: job.AbortOnErr,
		},
	}
}
