package vault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/cloudradar-monitoring/rport/share/enc"

	chshare "github.com/cloudradar-monitoring/rport/share"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

var supportedFields = map[string]bool{
	"id":         true,
	"client_id":  true,
	"created_by": true,
	"created_at": true,
	"key":        true,
}

var WrongPasswordError = errors2.APIError{
	Message: "wrong password provided",
	Code:    http.StatusUnauthorized,
}

type Config interface {
	GetVaultDBPath() string
}

type UserDataProvider interface {
	GetGroups() []string
	GetUsername() string
}

type DbProvider interface {
	GetStatus(ctx context.Context) (DbStatus, error)
	SetStatus(ctx context.Context, newStatus DbStatus) error
	GetByID(ctx context.Context, id int) (val StoredValue, found bool, err error)
	List(ctx context.Context, lo *query.ListOptions) ([]ValueKey, error)
	FindByKeyAndClientID(ctx context.Context, key, clientID string) (val StoredValue, found bool, err error)
	Save(ctx context.Context, user string, idToUpdate int64, val *InputValue, nowDate time.Time) (int64, error)
	Delete(ctx context.Context, id int) error
	io.Closer
}

type PassManager interface {
	ValidatePass(passToCheck string) error
	PassMatch(dbStatus DbStatus, passToCheck string) (bool, error)
	GetEncRandValue(pass string) (encValue, decValue string, err error)
}

type DbProviderFactory interface {
	GetDbProvider() DbProvider
	Init() error
}

type Manager struct {
	passLock  sync.RWMutex
	pass      string
	dbFactory DbProviderFactory
	pm        PassManager
	logger    *chshare.Logger
}

func NewManager(dbFactory DbProviderFactory, pm PassManager, logger *chshare.Logger) *Manager {
	return &Manager{
		passLock:  sync.RWMutex{},
		dbFactory: dbFactory,
		pm:        pm,
		logger:    logger,
	}
}

func (m *Manager) Init(ctx context.Context, pass string) error {
	if err := m.pm.ValidatePass(pass); err != nil {
		return err
	}

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

	err = m.dbFactory.Init()
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

	db := m.dbFactory.GetDbProvider()

	err = db.SetStatus(ctx, dbStatus)
	if err != nil {
		return err
	}

	m.passLock.Lock()
	defer m.passLock.Unlock()
	m.pass = pass
	m.logger.Infof("unlocked vault")

	return nil
}

func (m *Manager) isDatabaseInitialized(ctx context.Context) (bool, error) {
	db := m.dbFactory.GetDbProvider()

	dbStatus, err := db.GetStatus(ctx)
	if err != nil {
		if errors.Is(err, ErrDatabaseNotInitialised) {
			return false, nil
		}
		return false, err
	}

	if dbStatus.StatusName == DbStatusInit {
		return true, nil
	}

	return false, nil
}

func (m *Manager) UnLock(ctx context.Context, pass string) error {
	if !m.IsLocked() {
		return errors2.APIError{
			Message: "vault is already unlocked",
			Code:    http.StatusConflict,
		}
	}

	db := m.dbFactory.GetDbProvider()

	dbStatus, err := db.GetStatus(ctx)
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

	m.passLock.Lock()
	defer m.passLock.Unlock()

	m.pass = pass

	return nil
}

func (m *Manager) Lock(ctx context.Context) error {
	if m.IsLocked() {
		return errors2.APIError{
			Message: "vault is already locked",
			Code:    http.StatusConflict,
		}
	}

	db := m.dbFactory.GetDbProvider()
	dbStatus, err := db.GetStatus(ctx)
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
	m.passLock.Lock()
	defer m.passLock.Unlock()

	m.pass = ""

	return nil
}

func (m *Manager) IsLocked() bool {
	m.passLock.RLock()
	defer m.passLock.RUnlock()

	return m.pass == ""
}

func (m *Manager) Status(ctx context.Context) (StatusReport, error) {
	sr := StatusReport{}

	db := m.dbFactory.GetDbProvider()
	dbStatus, err := db.GetStatus(ctx)
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

	listOptions := query.ConvertGetParamsToFilterOptions(re)

	err = query.ValidateListOptions(listOptions, supportedFields)
	if err != nil {
		return nil, err
	}

	db := m.dbFactory.GetDbProvider()

	return db.List(ctx, listOptions)
}

func (m *Manager) checkGroupAccess(val *StoredValue, user UserDataProvider) error {
	if val == nil || val.RequiredGroup == "" {
		return nil
	}
	userGroupMatches := false
	for _, gr := range user.GetGroups() {
		if gr != val.RequiredGroup {
			continue
		}
		userGroupMatches = true
		break
	}
	if !userGroupMatches {
		return errors2.APIError{
			Message: "your group doesn't allow access to this value",
			Code:    http.StatusForbidden,
		}
	}

	return nil
}

func (m *Manager) GetOne(ctx context.Context, id int, user UserDataProvider) (StoredValue, bool, error) {
	err := m.checkUnlockedAndInitialized(ctx)
	if err != nil {
		return StoredValue{}, false, err
	}

	db := m.dbFactory.GetDbProvider()

	val, found, err := db.GetByID(ctx, id)
	if err != nil {
		return StoredValue{}, false, err
	}

	if !found {
		return StoredValue{}, false, nil
	}

	err = m.checkGroupAccess(&val, user)
	if err != nil {
		return StoredValue{}, false, err
	}

	m.passLock.RLock()
	defer m.passLock.RUnlock()

	decryptedValue, err := enc.Aes256DecryptByPassFromBase64String(val.Value, m.pass)
	if err != nil {
		return StoredValue{}, false, err
	}
	val.Value = string(decryptedValue)

	return val, true, nil
}

func (m *Manager) Store(ctx context.Context, existingID int64, valueToStore *InputValue, user UserDataProvider) (StoredValueID, error) {
	err := m.checkUnlockedAndInitialized(ctx)
	if err != nil {
		return StoredValueID{}, err
	}

	err = Validate(valueToStore)
	if err != nil {
		return StoredValueID{}, err
	}

	db := m.dbFactory.GetDbProvider()

	storedValue, found, err := db.FindByKeyAndClientID(ctx, valueToStore.Key, valueToStore.ClientID)
	if err != nil {
		return StoredValueID{}, err
	}

	if existingID > 0 {
		val, found2, err := db.GetByID(ctx, int(existingID))
		if err != nil {
			return StoredValueID{}, err
		}

		if !found2 {
			return StoredValueID{}, errors2.APIError{
				Message: "cannot find entry by the provided existingID",
				Code:    http.StatusNotFound,
			}
		}

		err = m.checkGroupAccess(&val, user)
		if err != nil {
			return StoredValueID{}, err
		}
	}

	if found && (existingID == 0 || storedValue.ID != int(existingID)) {
		return StoredValueID{}, errors2.APIError{
			Message: fmt.Sprintf("another key '%s' exists for this client '%s'", valueToStore.Key, valueToStore.ClientID),
			Code:    http.StatusConflict,
		}
	}

	m.passLock.RLock()
	defer m.passLock.RUnlock()

	encValue, err := enc.Aes256EncryptByPassToBase64String([]byte(valueToStore.Value), m.pass)
	if err != nil {
		return StoredValueID{}, err
	}

	valueToStore.Value = encValue

	res := StoredValueID{}
	res.ID, err = db.Save(ctx, user.GetUsername(), existingID, valueToStore, time.Now())
	if err != nil {
		return StoredValueID{}, err
	}

	return res, nil
}

func (m *Manager) Delete(ctx context.Context, id int, user UserDataProvider) error {
	err := m.checkUnlockedAndInitialized(ctx)
	if err != nil {
		return err
	}

	db := m.dbFactory.GetDbProvider()

	storedValue, found, err := db.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !found {
		return errors2.APIError{
			Message: "cannot find this entry by the provided id",
			Code:    http.StatusNotFound,
		}
	}

	err = m.checkGroupAccess(&storedValue, user)
	if err != nil {
		return err
	}

	err = db.Delete(ctx, id)
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

func (m *Manager) Close() error {
	db := m.dbFactory.GetDbProvider()
	return db.Close()
}
