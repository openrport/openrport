package schedule

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/validation"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/query"
	"github.com/cloudradar-monitoring/rport/share/random"
)

var supportedSortAndFilters = map[string]bool{
	"id":         true,
	"created_at": true,
	"created_by": true,
	"name":       true,
	"type":       true,
	"client_id":  true,
	"group_id":   true,
}

type Provider interface {
	Insert(context.Context, *Schedule) error
	Update(context.Context, *Schedule) error
	List(context.Context, *query.ListOptions) ([]*Schedule, error)
	Count(context.Context, *query.ListOptions) (int, error)
	Get(context.Context, string) (*Schedule, error)
	Delete(context.Context, string) error
}

type Cron interface {
	Validate(string) error
	Add(string, string, func(string)) error
	Remove(string)
}

type Manager struct {
	*logger.Logger
	provider Provider
	cron     Cron
}

func New(ctx context.Context, logger *logger.Logger, db *sqlx.DB) (*Manager, error) {
	m := &Manager{
		Logger:   logger,
		provider: newSQLiteProvider(db),
		cron:     newCron(),
	}

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

func (m *Manager) List(ctx context.Context, r *http.Request) (*api.SuccessPayload, error) {
	listOptions := query.GetListOptions(r)

	err := query.ValidateListOptions(listOptions, supportedSortAndFilters, supportedSortAndFilters, nil /*fields*/, &query.PaginationConfig{
		MaxLimit:     100,
		DefaultLimit: 20,
	})
	if err != nil {
		return nil, err
	}

	entries, err := m.provider.List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	count, err := m.provider.Count(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	return &api.SuccessPayload{
		Data: entries,
		Meta: api.NewMeta(count),
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
			Err:        fmt.Errorf("Type must be 'command' or 'script'"),
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

	return nil
}

func (m *Manager) addCron(s *Schedule) error {
	return m.cron.Add(s.ID, s.Schedule, m.run)
}

func (m *Manager) run(id string) {
	m.Infof("Running cron: %s", id)
}
