package vault

import (
	"context"
	"net/http"
	"sync"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

const defaultDBName = "vault.sqlite3"

var WrongPasswordError = errors2.APIError{
	Message: "wrong password provided",
	Code:    http.StatusUnauthorized,
}

type Config interface {
	GetDatabasePath() string
}

type DbProvider interface {
	Init(ctx context.Context) error
	GetStatus(ctx context.Context) (DbStatus, error)
	SetStatus(ctx context.Context, newStatus DbStatus) error
}

type PassManager interface {
	ValidatePass(passToCheck string) error
	PassMatch(dbStatus DbStatus, passToCheck string) (bool, error)
	GetEncRandValue(pass string) (encValue, decValue string, err error)
}

type Manager struct {
	passLock sync.RWMutex
	pass     string
	db       DbProvider
	pm       PassManager
}

func NewManager(db DbProvider, pm PassManager) *Manager {
	return &Manager{
		passLock: sync.RWMutex{},
		db:       db,
		pm:       pm,
	}
}

func (m *Manager) Init(ctx context.Context, pass string) error {
	if err := m.pm.ValidatePass(pass); err != nil {
		return err
	}
	m.passLock.Lock()
	defer m.passLock.Unlock()

	err := m.db.Init(ctx)
	if err != nil {
		return err
	}

	dbStatus, err := m.db.GetStatus(ctx)
	if err != nil {
		return err
	}
	if dbStatus.StatusName == DbStatusInit {
		return errors2.APIError{
			Message: "vault is already initialized",
			Code:    http.StatusConflict,
		}
	}

	dbStatus.StatusName = DbStatusInit
	dbStatus.EncCheckValue, dbStatus.DecCheckValue, err = m.pm.GetEncRandValue(pass)
	if err != nil {
		return err
	}

	err = m.db.SetStatus(ctx, dbStatus)
	if err != nil {
		return err
	}

	m.pass = pass

	return nil
}

func (m *Manager) UnLock(ctx context.Context, pass string) error {
	m.passLock.Lock()
	defer m.passLock.Unlock()

	if !m.IsLocked() {
		return errors2.APIError{
			Message: "vault is already unlocked",
			Code:    http.StatusConflict,
		}
	}

	dbStatus, err := m.db.GetStatus(ctx)
	if err != nil {
		return err
	}
	if dbStatus.StatusName == "" || dbStatus.StatusName == DbStatusNotInit {
		return errors2.APIError{
			Message: "vault is not yet initialized",
			Code:    http.StatusConflict,
		}
	}

	passMatch, err := m.pm.PassMatch(dbStatus, pass)
	if err != nil {
		return err
	}

	if !passMatch {
		return WrongPasswordError
	}

	m.pass = pass

	return nil
}

func (m *Manager) Lock(ctx context.Context) error {
	m.passLock.Lock()
	defer m.passLock.Unlock()

	if m.IsLocked() {
		return errors2.APIError{
			Message: "vault is already locked",
			Code:    http.StatusConflict,
		}
	}

	dbStatus, err := m.db.GetStatus(ctx)
	if err != nil {
		return err
	}
	if dbStatus.StatusName == "" || dbStatus.StatusName == DbStatusNotInit {
		return errors2.APIError{
			Message: "vault is not yet initialized",
			Code:    http.StatusConflict,
		}
	}

	m.pass = ""

	return nil
}

func (m *Manager) IsLocked() bool {
	return m.pass == ""
}

func (m *Manager) Status(ctx context.Context) (StatusReport, error) {
	m.passLock.Lock()
	defer m.passLock.Unlock()

	sr := StatusReport{}

	dbStatus, err := m.db.GetStatus(ctx)
	if err != nil {
		return sr, err
	}

	if dbStatus.StatusName == "" {
		dbStatus.StatusName = DbStatusNotInit
	}

	sr.InitStatus = dbStatus.StatusName

	if dbStatus.StatusName == DbStatusNotInit {
		return sr, nil
	}

	if m.IsLocked() {
		sr.LockStatus = StatusLocked
	} else {
		sr.LockStatus = StatusUnlocked
	}

	return sr, nil
}
