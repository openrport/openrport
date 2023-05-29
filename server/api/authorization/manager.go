package authorization

import (
	"context"
	"fmt"
	"io"
	"net/http"

	errors2 "github.com/realvnc-labs/rport/server/api/errors"
)

type DbProvider interface {
	Get(ctx context.Context, username, prefix string) (*APIToken, error)
	GetByUserAndTokenName(ctx context.Context, username, name string) (*APIToken, error)
	GetAllForUser(ctx context.Context, username string) ([]*APIToken, error)
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
	return m.db.Get(ctx, username, prefix)
}

func (m *Manager) GetAll(ctx context.Context, username string) ([]*APIToken, error) {
	val, err := m.db.GetAllForUser(ctx, username)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (m *Manager) Create(ctx context.Context, tokenLine *APIToken) error {
	err := checkNameExist(ctx, m.db, tokenLine.Username, tokenLine.Name)
	if err != nil {
		return err
	}

	err = m.db.Save(ctx, tokenLine)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) Save(ctx context.Context, tokenLine *APIToken) error {
	err := checkNameExist(ctx, m.db, tokenLine.Username, tokenLine.Name)
	if err != nil {
		return err
	}

	err = m.db.Save(ctx, tokenLine)
	if err != nil {
		return err
	}

	return nil
}

func checkNameExist(ctx context.Context, db DbProvider, username, name string) error {
	if name != "" {
		item, err := db.GetByUserAndTokenName(ctx, username, name)
		if err != nil {
			return err
		}
		if item != nil {
			return fmt.Errorf("A token with the same name already exists")
		}
	}
	return nil
}

func (m *Manager) Delete(ctx context.Context, username, prefix string) error {
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
