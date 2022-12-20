package authorization

import (
	"context"
	"io"
	"net/http"
	"time"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

var (
	supportedSortAndFilters = map[string]bool{
		"username":   true,
		"prefix":     true,
		"created_at": true,
		"expires_at": true,
		"scope":      true,
		"token":      true,
	}
	supportedFields = map[string]map[string]bool{
		"APIToken": {
			"username":   true,
			"prefix":     true,
			"created_at": true,
			"expires_at": true,
			"scope":      true,
			"token":      true,
		},
	}
	manualFiltersConfig = map[string]bool{
		"tags": true,
	}
)

type DbProvider interface {
	Get(ctx context.Context, username, prefix string) (*APIToken, error)
	GetAll(ctx context.Context, username string) ([]*APIToken, error)
	Save(ctx context.Context, tokenLine *APIToken) error
	Delete(ctx context.Context, username, prefix string) error
	io.Closer
}

type Manager struct {
	db DbProvider
}

func NewManager(db DbProvider) *Manager {
	return &Manager{
		db: db,
	}
}

func (m *Manager) Get(ctx context.Context, username, prefix string) (*APIToken, error) {
	val, err := m.db.Get(ctx, username, prefix)
	if err != nil {
		return nil, err
	}

	return val, nil
}
func (m *Manager) GetAll(ctx context.Context, username string) ([]*APIToken, error) {
	val, err := m.db.GetAll(ctx, username)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (m *Manager) Save(ctx context.Context, tokenLine *APIToken) error {
	// existingAPIToken, err := m.db.List(ctx, &query.ListOptions{
	// 	Filters: []query.FilterOption{
	// 		{
	// 			Column: []string{"scope"},
	// 			Values: []string{valueToStore.Scope},
	// 		},
	// 	},
	// })
	// if err != nil {
	// 	return nil, err
	// }
	// if len(existingAPIToken) > 0 {
	// 	return nil, errors2.APIError{
	// 		Message:    fmt.Sprintf("another APIToken with the same prefix '%s' exists", valueToStore.Scope),
	// 		HTTPStatus: http.StatusConflict,
	// 	}
	// }

	now := time.Now()
	tokenLine.CreatedAt = &now

	err := m.db.Save(ctx, tokenLine)
	if err != nil {
		return err
	}

	return nil
}

/*
	func (m *Manager) Update(ctx context.Context, existingID string, valueToStore *APIToken, username string) (*APIToken, error) {
		existing, foundByID, err := m.db.GetByID(ctx, existingID, &query.RetrieveOptions{})
		if err != nil {
			return nil, err
		}

		if !foundByID {
			return nil, errors2.APIError{
				Message:    "cannot find entry by the provided ID",
				HTTPStatus: http.StatusNotFound,
			}
		}

		APITokensWithSameName, err := m.db.List(ctx, &query.ListOptions{
			Filters: []query.FilterOption{
				{
					Column: []string{"name"},
					Values: []string{valueToStore.Name},
				},
			},
		})
		if err != nil {
			return nil, err
		}

		if len(APITokensWithSameName) > 0 && APITokensWithSameName[0].ID != existingID {
			return nil, errors2.APIError{
				Message:    fmt.Sprintf("another APIToken with the same name '%s' exists", valueToStore.Name),
				HTTPStatus: http.StatusConflict,
			}
		}
		if valueToStore.TimoutSec == 0 {
			valueToStore.TimoutSec = DefaultTimeoutSec
		}

		now := time.Now()
		APITokenToSave := &APIToken{
			ID:        existingID,
			Name:      valueToStore.Name,
			CreatedBy: existing.CreatedBy,
			CreatedAt: existing.CreatedAt,
			UpdatedBy: username,
			UpdatedAt: &now,
			Cmd:       valueToStore.Cmd,
			Tags:      (*types.StringSlice)(&valueToStore.Tags),
			TimoutSec: &valueToStore.TimoutSec,
		}
		_, err = m.db.Save(ctx, APITokenToSave)
		if err != nil {
			return nil, err
		}

		return APITokenToSave, nil
	}
*/
func (m *Manager) Delete(ctx context.Context, username, prefix string) error {
	// _, found, err := m.db.GetByID(ctx, id, &query.RetrieveOptions{})
	// if err != nil {
	// 	return errors2.APIError{
	// 		Err:        err,
	// 		HTTPStatus: http.StatusInternalServerError,
	// 	}
	// }

	// if !found {
	// 	return errors2.APIError{
	// 		Message:    "cannot find this APIToken by the provided prefix",
	// 		HTTPStatus: http.StatusNotFound,
	// 	}
	// }

	err := m.db.Delete(ctx, username, prefix)
	if err != nil {
		return errors2.APIError{
			Err:        err,
			HTTPStatus: http.StatusInternalServerError,
		}
	}

	return nil
}

func (m *Manager) Close() error {
	return m.db.Close()
}
