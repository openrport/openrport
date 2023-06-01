package schedule

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/server/api/jobs"
	"github.com/realvnc-labs/rport/server/validation"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/random"
)

var (
	supportedSorts = map[string]bool{
		"id":         true,
		"created_at": true,
		"created_by": true,
		"name":       true,
		"type":       true,
	}
	supportedFilters = map[string]bool{
		"id":         true,
		"created_at": true,
		"created_by": true,
		"name":       true,
		"type":       true,
		"client_ids": true,
		"group_ids":  true,
	}
	manualFiltersConfig = map[string]bool{
		"client_ids": true,
		"group_ids":  true,
	}
)

type Provider interface {
	Insert(context.Context, *Schedule) error
	Update(context.Context, *Schedule) error
	List(context.Context, *query.ListOptions) ([]*Schedule, error)
	Get(context.Context, string) (*Schedule, error)
	Delete(context.Context, string) error
	CountJobsInProgress(ctx context.Context, scheduleID string, timeoutSec int) (int, error)
}

type Cron interface {
	Validate(string) error
	Add(string, string, func(context.Context, string)) error
	Remove(string)
}

type JobRunner interface {
	StartMultiClientJob(ctx context.Context, multiJobRequest *jobs.MultiJobRequest) (*models.MultiJob, error)
}

type Manager struct {
	*logger.Logger
	jobRunner JobRunner
	provider  Provider
	cron      Cron

	runRemoteCmdTimeoutSec int
}

func New(ctx context.Context, logger *logger.Logger, db *sqlx.DB, jobRunner JobRunner, runRemoteCmdTimeoutSec int) (*Manager, error) {
	m := NewManager(jobRunner, db, logger, runRemoteCmdTimeoutSec)

	existing, err := m.provider.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, cron := range existing {
		err := m.addCron(cron)
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}

func NewManager(jobRunner JobRunner, db *sqlx.DB, logger *logger.Logger, runRemoteCmdTimeoutSec int) (m *Manager) {
	m = &Manager{
		Logger:    logger,
		jobRunner: jobRunner,
		provider:  newSQLiteProvider(db),
		cron:      newCron(),

		runRemoteCmdTimeoutSec: runRemoteCmdTimeoutSec,
	}
	return m
}

func (m *Manager) List(ctx context.Context, r *http.Request) (*api.SuccessPayload, error) {
	listOptions := query.GetListOptions(r)

	err := query.ValidateListOptions(listOptions, supportedSorts, supportedFilters, nil /*fields*/, &query.PaginationConfig{
		MaxLimit:     100,
		DefaultLimit: 20,
	})
	if err != nil {
		return nil, err
	}

	manualFilters, dbFilters := query.SplitFilters(listOptions.Filters, manualFiltersConfig)
	pagination := listOptions.Pagination

	listOptions.Filters = dbFilters
	listOptions.Pagination = nil

	entries, err := m.provider.List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	filtered := make([]*Schedule, 0, len(entries))
	for _, entry := range entries {
		matches, err := query.MatchesFilters(entry, manualFilters)
		if err != nil {
			return nil, err
		}
		if matches {
			filtered = append(filtered, entry)
		}
	}

	totalCount := len(filtered)
	start, end := pagination.GetStartEnd(totalCount)
	limited := filtered[start:end]

	return &api.SuccessPayload{
		Data: limited,
		Meta: api.NewMeta(totalCount),
	}, nil
}

func (m *Manager) Get(ctx context.Context, id string) (*Schedule, error) {
	return m.provider.Get(ctx, id)
}

func (m *Manager) Create(ctx context.Context, s *Schedule, user string) (*Schedule, error) {
	var err error
	s.ID, err = random.UUID4()
	if err != nil {
		return nil, err
	}
	s.CreatedAt = time.Now()
	s.CreatedBy = user

	err = m.validate(s)
	if err != nil {
		return nil, err
	}

	err = m.provider.Insert(ctx, s)
	if err != nil {
		return nil, err
	}

	err = m.addCron(s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (m *Manager) Update(ctx context.Context, id string, s *Schedule) (*Schedule, error) {
	s.ID = id

	err := m.validate(s)
	if err != nil {
		return nil, err
	}

	err = m.provider.Update(ctx, s)
	if err != nil {
		return nil, err
	}

	m.cron.Remove(s.ID)
	err = m.addCron(s)
	if err != nil {
		return nil, err
	}

	s, err = m.provider.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	err := m.provider.Delete(ctx, id)
	if err != nil {
		return err
	}

	m.cron.Remove(id)
	return nil
}

func (m *Manager) validate(s *Schedule) error {
	if s.Type != TypeCommand && s.Type != TypeScript {
		return &errors.APIError{
			Message:    "Invalid type.",
			Err:        fmt.Errorf("type must be 'command' or 'script'"),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	err := m.cron.Validate(s.Schedule)
	if err != nil {
		return &errors.APIError{
			Message:    "Invalid schedule.",
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		}
	}

	err = validation.ValidateInterpreter(s.Details.Interpreter, s.Type == TypeScript)
	if err != nil {
		return &errors.APIError{
			Message:    "Invalid interpreter.",
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		}
	}

	switch s.Type {
	case TypeCommand:
		if s.Details.Command == "" {
			return &errors.APIError{
				Message:    "Empty command.",
				Err:        fmt.Errorf("command cannot be empty"),
				HTTPStatus: http.StatusBadRequest,
			}
		}
	case TypeScript:
		if s.Details.Script == "" {
			return &errors.APIError{
				Message:    "Empty script.",
				Err:        fmt.Errorf("script cannot be empty"),
				HTTPStatus: http.StatusBadRequest,
			}
		}
		_, err := base64.StdEncoding.DecodeString(s.Details.Script)
		if err != nil {
			return &errors.APIError{
				Message:    "Invalid script.",
				Err:        err,
				HTTPStatus: http.StatusBadRequest,
			}
		}
	}

	return nil
}

func (m *Manager) addCron(s *Schedule) error {
	return m.cron.Add(s.ID, s.Schedule, m.run)
}

func (m *Manager) run(ctx context.Context, id string) {
	schedule, err := m.provider.Get(ctx, id)
	if err != nil {
		m.Errorf("Could not get schedule %s: %v", id, err)
		return
	}
	if schedule == nil {
		// schedule not found in db, probably deleted by user
		return
	}

	if !schedule.Details.Overlaps {
		timeoutSec := schedule.Details.TimeoutSec
		if timeoutSec <= 0 {
			timeoutSec = m.runRemoteCmdTimeoutSec
		}
		cnt, err := m.provider.CountJobsInProgress(ctx, id, timeoutSec)
		if err != nil {
			m.Errorf("Could not count jobs in progress for schedule %s: %v", id, err)
			return
		}
		if cnt > 0 {
			m.Infof("Skipping non-overlapping schedule %s, because it has jobs in progress.", id)
			return
		}
	}

	m.Infof("Running schedule: %s", id)

	_, err = m.jobRunner.StartMultiClientJob(ctx, &jobs.MultiJobRequest{
		ScheduleID:          &schedule.ID,
		Username:            schedule.CreatedBy,
		ClientIDs:           schedule.Details.ClientIDs,
		ClientTags:          schedule.Details.ClientTags,
		GroupIDs:            schedule.Details.GroupIDs,
		Command:             schedule.Details.Command,
		Script:              schedule.Details.Script,
		Cwd:                 schedule.Details.Cwd,
		IsSudo:              schedule.Details.IsSudo,
		Interpreter:         schedule.Details.Interpreter,
		TimeoutSec:          schedule.Details.TimeoutSec,
		ExecuteConcurrently: schedule.Details.ExecuteConcurrently,
		AbortOnError:        schedule.Details.AbortOnError,
		IsScript:            schedule.Type == TypeScript,
	})
	if err != nil {
		m.Errorf("Error running schedule %s: %v", id, err)
		return
	}
}
