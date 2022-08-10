package jobs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

var MultiJobSupportedFilters = map[string]bool{
	"jid":               true,
	"started_at[gt]":    true,
	"started_at[lt]":    true,
	"started_at[since]": true,
	"started_at[until]": true,
	"created_by":        true,
	"schedule_id":       true,
}
var MultiJobSupportedSorts = map[string]bool{
	"jid":         true,
	"started_at":  true,
	"created_by":  true,
	"schedule_id": true,
}

// GetMultiJob returns a multi-client job with fetched all clients' jobs.
func (p *SqliteProvider) GetMultiJob(ctx context.Context, jid string) (*models.MultiJob, error) {
	res := &multiJobSqlite{}
	err := p.db.Get(res, "SELECT * FROM multi_jobs WHERE jid=?", jid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	multiJob := res.convert()

	options := &query.ListOptions{
		Filters: []query.FilterOption{
			{Column: []string{"multi_job_id"}, Values: []string{jid}},
		},
		Sorts: []query.SortOption{
			{Column: "started_at", IsASC: false},
			{Column: "jid", IsASC: true},
		},
		Pagination: query.NewPagination(DefaultLimit, 0),
	}
	jobs, err := p.List(ctx, options)
	if err != nil {
		return nil, err
	}
	multiJob.Jobs = jobs

	return multiJob, nil
}

// GetMultiJobSummaries returns a list of summaries of multi-clients jobs filtered by options and sorted by started_at(desc), jid order.
func (p *SqliteProvider) GetMultiJobSummaries(ctx context.Context, options *query.ListOptions) ([]*models.MultiJobSummary, error) {
	var res []*multiJobSummarySqlite

	if len(options.Sorts) == 0 {
		options.Sorts = []query.SortOption{
			{
				Column: "DATETIME(started_at)",
				IsASC:  false,
			},
			{
				Column: "jid",
				IsASC:  true,
			},
		}
	}
	q := "SELECT jid, started_at, created_by, schedule_id FROM multi_jobs"
	q, params := p.converter.ConvertListOptionsToQuery(options, q)

	err := p.db.SelectContext(ctx, &res, q, params...)
	if err != nil {
		return nil, err
	}
	return convertMultiJSs(res), nil
}

// CountMultiJobs counts multi-clients jobs filtered by options
func (p *SqliteProvider) CountMultiJobs(ctx context.Context, options *query.ListOptions) (int, error) {
	var result int

	countOptions := *options
	countOptions.Pagination = nil
	q := "SELECT count(*) FROM multi_jobs"
	q, params := p.converter.ConvertListOptionsToQuery(&countOptions, q)

	err := p.db.GetContext(ctx, &result, q, params...)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// SaveMultiJob creates a new or updates an existing multi-client job (without child jobs).
func (p *SqliteProvider) SaveMultiJob(job *models.MultiJob) error {
	_, err := p.db.NamedExec(`
INSERT OR REPLACE INTO multi_jobs (
	jid, started_at, created_by, schedule_id, details
) VALUES (
	:jid, :started_at, :created_by, :schedule_id, :details
)`,
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
	JID        string    `db:"jid"`
	StartedAt  time.Time `db:"started_at"`
	CreatedBy  string    `db:"created_by"`
	ScheduleID *string   `db:"schedule_id"`
}

type multiJobDetailSqlite struct {
	ClientIDs   []string              `json:"client_ids"`
	GroupIDs    []string              `json:"group_ids"`
	ClientTags  *models.JobClientTags `json:"tags"`
	Command     string                `json:"command"`
	Interpreter string                `json:"interpreter"`
	Cwd         string                `json:"cwd"`
	IsSudo      bool                  `json:"is_sudo"`
	TimeoutSec  int                   `json:"timeout_sec"`
	Concurrent  bool                  `json:"concurrent"`
	AbortOnErr  bool                  `json:"abort_on_err"`
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
		JID:        js.JID,
		StartedAt:  js.StartedAt,
		CreatedBy:  js.CreatedBy,
		ScheduleID: js.ScheduleID,
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
		GroupIDs:        d.GroupIDs,
		ClientTags:      d.ClientTags,
		Command:         d.Command,
		Cwd:             d.Cwd,
		IsSudo:          d.IsSudo,
		Interpreter:     d.Interpreter,
		TimeoutSec:      d.TimeoutSec,
		Concurrent:      d.Concurrent,
		AbortOnErr:      d.AbortOnErr,
	}
}

func convertMultiJobToSqlite(job *models.MultiJob) *multiJobSqlite {
	return &multiJobSqlite{
		multiJobSummarySqlite: multiJobSummarySqlite{
			JID:        job.JID,
			StartedAt:  job.StartedAt,
			CreatedBy:  job.CreatedBy,
			ScheduleID: job.ScheduleID,
		},
		Details: &multiJobDetailSqlite{
			ClientIDs:   job.ClientIDs,
			GroupIDs:    job.GroupIDs,
			ClientTags:  job.ClientTags,
			Command:     job.Command,
			Interpreter: job.Interpreter,
			Cwd:         job.Cwd,
			IsSudo:      job.IsSudo,
			TimeoutSec:  job.TimeoutSec,
			Concurrent:  job.Concurrent,
			AbortOnErr:  job.AbortOnErr,
		},
	}
}
