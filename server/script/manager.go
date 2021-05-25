package script

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudradar-monitoring/rport/share/query"

	chshare "github.com/cloudradar-monitoring/rport/share"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

var supportedFields = map[string]bool{
	"id":         true,
	"name":       true,
	"created_by": true,
	"created_at": true,
}

type DbProvider interface {
	GetByID(ctx context.Context, id string) (val *Script, found bool, err error)
	List(ctx context.Context, lo *query.ListOptions) ([]Script, error)
	Save(ctx context.Context, s *Script, nowDate time.Time) (string, error)
	Delete(ctx context.Context, id string) error
	io.Closer
}

type UserDataProvider interface {
	GetUsername() string
}

type Manager struct {
	db     DbProvider
	logger *chshare.Logger
}

func NewManager(db DbProvider, logger *chshare.Logger) *Manager {
	return &Manager{
		db:     db,
		logger: logger,
	}
}

func (m *Manager) List(ctx context.Context, re *http.Request) ([]Script, error) {
	listOptions := query.ConvertGetParamsToFilterOptions(re)

	err := m.validateListOptions(listOptions)
	if err != nil {
		return nil, err
	}

	return m.db.List(ctx, listOptions)
}

func (m *Manager) validateListOptions(lo *query.ListOptions) error {
	errs := errors2.APIErrors{}
	for i := range lo.Sorts {
		ok := supportedFields[lo.Sorts[i].Column]
		if !ok {
			errs = append(errs, errors2.APIError{
				Message: fmt.Sprintf("unsupported sort field '%s'", lo.Sorts[i].Column),
				Code:    http.StatusBadRequest,
			})
		}
	}

	for i := range lo.Filters {
		ok := supportedFields[lo.Filters[i].Column]
		if !ok {
			errs = append(errs, errors2.APIError{
				Message: fmt.Sprintf("unsupported filter field '%s'", lo.Filters[i].Column),
				Code:    http.StatusBadRequest,
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func (m *Manager) GetOne(ctx context.Context, id string) (*Script, bool, error) {
	val, found, err := m.db.GetByID(ctx, id)
	if err != nil {
		return nil, false, err
	}

	if !found {
		return nil, false, nil
	}

	return val, true, nil
}

func (m *Manager) Store(ctx context.Context, existingID string, valueToStore *InputScript, userProvider UserDataProvider) (*Script, error) {
	err := Validate(valueToStore)
	if err != nil {
		return nil, err
	}

	existingScript, err := m.db.List(ctx, &query.ListOptions{
		Filters: []query.FilterOption{
			{
				Column: "name",
				Values: []string{valueToStore.Name},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if existingID != "" {
		_, foundByID, err := m.db.GetByID(ctx, existingID)
		if err != nil {
			return nil, err
		}

		if !foundByID {
			return nil, errors2.APIError{
				Message: "cannot find entry by the provided ID",
				Code:    http.StatusNotFound,
			}
		}
	}

	if len(existingScript) > 0 && (existingID == "" || existingScript[0].ID != existingID) {
		return nil, errors2.APIError{
			Message: fmt.Sprintf("another script with the same name '%s' exists", valueToStore.Name),
			Code:    http.StatusConflict,
		}
	}

	now := time.Now()
	scriptToSave := &Script{
		ID:          existingID,
		Name:        valueToStore.Name,
		CreatedBy:   userProvider.GetUsername(),
		CreatedAt:   now,
		Interpreter: valueToStore.Interpreter,
		IsSudo:      valueToStore.IsSudo,
		Cwd:         valueToStore.Cwd,
		Script:      valueToStore.Script,
	}
	scriptToSave.ID, err = m.db.Save(ctx, scriptToSave, now)
	if err != nil {
		return nil, err
	}

	return scriptToSave, nil
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	_, found, err := m.db.GetByID(ctx, id)
	if err != nil {
		return errors2.APIError{
			Err:  err,
			Code: http.StatusInternalServerError,
		}
	}

	if !found {
		return errors2.APIError{
			Message: "cannot find this entry by the provided id",
			Code:    http.StatusNotFound,
		}
	}

	err = m.db.Delete(ctx, id)
	if err != nil {
		return errors2.APIError{
			Err:  err,
			Code: http.StatusInternalServerError,
		}
	}

	return nil
}

func (m *Manager) Close() error {
	return m.db.Close()
}
