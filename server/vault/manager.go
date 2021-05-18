package vault

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/cloudradar-monitoring/rport/share/enc"

	chshare "github.com/cloudradar-monitoring/rport/share"

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

type UserDataProvider interface {
	GetGroups() []string
}

type DbProvider interface {
	Init(ctx context.Context) error
	GetStatus(ctx context.Context) (DbStatus, error)
	SetStatus(ctx context.Context, newStatus DbStatus) error
	GetByID(ctx context.Context, id int) (val StoredValue, found bool, err error)
	List(ctx context.Context, lo *ListOptions) ([]ValueKey, error)
	FindByKeyAndClientID(ctx context.Context, key, clientID string) (val StoredValue, found bool, err error)
	Save(ctx context.Context, user string, idToUpdate int, val *InputValue, nowDate time.Time) error
	Delete(ctx context.Context, id int) error
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
	logger   *chshare.Logger
}

func NewManager(db DbProvider, pm PassManager, logger *chshare.Logger) *Manager {
	return &Manager{
		passLock: sync.RWMutex{},
		db:       db,
		pm:       pm,
		logger:   logger,
	}
}

func (m *Manager) Init(ctx context.Context, pass string) error {
	if err := m.pm.ValidatePass(pass); err != nil {
		return err
	}
	m.passLock.Lock()
	defer m.passLock.Unlock()

	isInit, err := m.isDatabaseInitialized(ctx)
	if err != nil {
		return err
	}

	if isInit {
		return errors2.APIError{
			Message: "vault is already initialized",
			Code:    http.StatusConflict,
		}
	}

	err = m.db.Init(ctx)
	if err != nil {
		return err
	}
	m.logger.Infof("initialized vault")

	dbStatus := DbStatus{
		StatusName: DbStatusInit,
	}
	dbStatus.EncCheckValue, dbStatus.DecCheckValue, err = m.pm.GetEncRandValue(pass)
	if err != nil {
		return err
	}

	err = m.db.SetStatus(ctx, dbStatus)
	if err != nil {
		return err
	}

	m.pass = pass
	m.logger.Infof("unlocked vault")

	return nil
}

func (m *Manager) isDatabaseInitialized(ctx context.Context) (bool, error) {
	dbStatus, err := m.db.GetStatus(ctx)
	if err != nil && !errors.Is(err, ErrDatabaseNotInitialised) {
		return false, err
	}

	if err != nil && errors.Is(err, ErrDatabaseNotInitialised) {
		return false, nil
	}

	if dbStatus.StatusName == DbStatusInit {
		return true, nil
	}

	return false, nil
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

	m.logger.Infof("unlocked vault")

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

	m.logger.Infof("locked vault")
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
	if err != nil && !errors.Is(err, ErrDatabaseNotInitialised) {
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

func (m *Manager) List(ctx context.Context, re *http.Request) ([]ValueKey, error) {
	err := m.checkUnlockedAndInitialized(ctx)
	if err != nil {
		return nil, err
	}

	listOptions := ConvertGetParamsToFilterOptions(re)

	return m.db.List(ctx, listOptions)
}

func (m *Manager) GetOne(ctx context.Context, id int, user UserDataProvider) (StoredValue, bool, error) {
	err := m.checkUnlockedAndInitialized(ctx)
	if err != nil {
		return StoredValue{}, false, err
	}

	val, found, err := m.db.GetByID(ctx, id)
	if err != nil {
		return StoredValue{}, false, err
	}

	if !found {
		return StoredValue{}, false, nil
	}

	if val.RequiredGroup != "" {
		userGroupMatches := false
		for _, gr := range user.GetGroups() {
			if gr != val.RequiredGroup {
				continue
			}
			userGroupMatches = true
			break
		}
		if !userGroupMatches {
			return StoredValue{}, false, errors2.APIError{
				Message: "your group doesn't allow access to this value",
				Code:    http.StatusForbidden,
			}
		}
	}

	decryptedValue, err := enc.Aes256DecryptByPassFromBase64String(val.Value, m.pass)
	if err != nil {
		return StoredValue{}, false, err
	}
	val.Value = string(decryptedValue)

	return val, true, nil
}

func (m *Manager) Store(ctx context.Context, id int, iv *InputValue, username string) error {
	err := m.checkUnlockedAndInitialized(ctx)
	if err != nil {
		return err
	}

	err = Validate(iv)
	if err != nil {
		return err
	}

	encValue, err := enc.Aes256EncryptByPassToBase64String([]byte(iv.Value), m.pass)
	if err != nil {
		return err
	}

	iv.Value = encValue

	err = m.db.Save(ctx, username, id, iv, time.Now())
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) Delete(ctx context.Context, id int) error {
	err := m.checkUnlockedAndInitialized(ctx)
	if err != nil {
		return err
	}

	err = m.db.Delete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) checkUnlockedAndInitialized(ctx context.Context) error {
	if m.IsLocked() {
		return errors2.APIError{
			Message: "vault is locked",
			Code:    http.StatusConflict,
		}
	}

	isInit, err := m.isDatabaseInitialized(ctx)
	if err != nil {
		return err
	}

	if !isInit {
		return errors2.APIError{
			Message: "vault is not initialized",
			Code:    http.StatusConflict,
		}
	}

	return nil
}
