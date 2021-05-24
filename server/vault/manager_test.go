package vault

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/share/enc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

type DbProviderMock struct {
	isInit  bool
	initErr error

	statusToGive     DbStatus
	statusToGiveErr  error
	statusToStore    DbStatus
	statusToStoreErr error

	getByID            int
	getByIDStoredValue StoredValue
	getByIDFound       bool
	getByIDError       error

	listOptionInput  *ListOptions
	listValuesToGive []ValueKey
	listErrorToGive  error

	findByKeyAndClientIDKey         string
	findByKeyAndClientIDClientID    string
	FindByKeyAndClientIDValueToGive StoredValue
	FindByKeyAndClientIDFoundToGive bool
	FindByKeyAndClientIDErrorToGive error

	SaveUserGiven    string
	SaveIDGiven      int64
	SaveIDToGive     int64
	SaveInputGiven   *InputValue
	SaveNowDateGiven time.Time
	SaveErrorToGive  error

	DeleteIDGiven     int
	DeleteErrorToGive error

	io.Closer
}

func (dpm *DbProviderMock) Init() error {
	dpm.isInit = true
	return dpm.initErr
}

func (dpm *DbProviderMock) GetStatus(ctx context.Context) (DbStatus, error) {
	return dpm.statusToGive, dpm.statusToGiveErr
}

func (dpm *DbProviderMock) SetStatus(ctx context.Context, newStatus DbStatus) error {
	dpm.statusToStore = newStatus
	return dpm.statusToStoreErr
}

func (dpm *DbProviderMock) GetByID(ctx context.Context, id int) (val StoredValue, found bool, err error) {
	dpm.getByID = id
	return dpm.getByIDStoredValue, dpm.getByIDFound, dpm.getByIDError
}

func (dpm *DbProviderMock) List(ctx context.Context, lo *ListOptions) ([]ValueKey, error) {
	dpm.listOptionInput = lo

	return dpm.listValuesToGive, dpm.listErrorToGive
}

func (dpm *DbProviderMock) FindByKeyAndClientID(ctx context.Context, key, clientID string) (val StoredValue, found bool, err error) {
	dpm.findByKeyAndClientIDKey = key
	dpm.findByKeyAndClientIDClientID = clientID

	return dpm.FindByKeyAndClientIDValueToGive, dpm.FindByKeyAndClientIDFoundToGive, dpm.FindByKeyAndClientIDErrorToGive
}

func (dpm *DbProviderMock) Save(ctx context.Context, user string, idToUpdate int64, val *InputValue, nowDate time.Time) (int64, error) {
	dpm.SaveUserGiven = user
	dpm.SaveIDGiven = idToUpdate
	dpm.SaveInputGiven = val
	dpm.SaveNowDateGiven = nowDate

	return dpm.SaveIDToGive, dpm.SaveErrorToGive
}

func (dpm *DbProviderMock) Delete(ctx context.Context, id int) error {
	dpm.DeleteIDGiven = id

	return dpm.DeleteErrorToGive
}

func (dpm *DbProviderMock) GetDbProvider() DbProvider {
	return dpm
}

type PassManagerMock struct {
	ValidatePassError error
	ValidatePassGiven string

	PassMatchDbStatusGiven DbStatus
	PassMatchPassGiven     string
	PassMatchToGive        bool
	PassMatchErr           error

	GetEncRandValuePassGiven      string
	GetEncRandValueEncValueToGive string
	GetEncRandValueDecValueToGive string
	GetEncRandValueErr            error
}

type UserDataProviderMock struct {
	GroupsToGive   []string
	UsernameToGive string
}

func (udpm UserDataProviderMock) GetGroups() []string {
	return udpm.GroupsToGive
}

func (udpm UserDataProviderMock) GetUsername() string {
	return udpm.UsernameToGive
}

func (pmm *PassManagerMock) ValidatePass(passToCheck string) error {
	pmm.ValidatePassGiven = passToCheck
	return pmm.ValidatePassError
}

func (pmm *PassManagerMock) PassMatch(dbStatus DbStatus, passToCheck string) (bool, error) {
	pmm.PassMatchDbStatusGiven = dbStatus
	pmm.PassMatchPassGiven = passToCheck

	return pmm.PassMatchToGive, pmm.PassMatchErr
}

func (pmm *PassManagerMock) GetEncRandValue(pass string) (encValue, decValue string, err error) {
	pmm.GetEncRandValuePassGiven = pass

	return pmm.GetEncRandValueEncValueToGive, pmm.GetEncRandValueDecValueToGive, pmm.GetEncRandValueErr
}

func TestManagerInit(t *testing.T) {
	dbProv := &DbProviderMock{}
	passManagerProv := &PassManagerMock{
		GetEncRandValueEncValueToGive: "123",
		GetEncRandValueDecValueToGive: "345",
	}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	const passToGive = "1234"
	err := mngr.Init(context.Background(), passToGive)
	require.NoError(t, err)

	assert.True(t, dbProv.isInit)
	assert.Equal(t, dbProv.statusToStore.StatusName, DbStatusInit)
	assert.Equal(t, dbProv.statusToStore.EncCheckValue, "123")
	assert.Equal(t, dbProv.statusToStore.DecCheckValue, "345")
	assert.False(t, mngr.IsLocked())
}

func TestManagerInitInvalidPassword(t *testing.T) {
	dbProv := &DbProviderMock{}
	passManagerProv := &PassManagerMock{
		ValidatePassError: errors.New("invalid password"),
	}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.Init(context.Background(), "12")
	assert.EqualError(t, err, "invalid password")
}

func TestManagerDbInitError(t *testing.T) {
	dbProv := &DbProviderMock{
		initErr: errors.New("failed to init database"),
	}
	passManagerProv := &PassManagerMock{}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.Init(context.Background(), "12")
	assert.EqualError(t, err, "failed to init database")
}

func TestManagerDbStatusReadError(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGiveErr: errors.New("failed to read status from database"),
	}
	passManagerProv := &PassManagerMock{}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.Init(context.Background(), "12")
	assert.EqualError(t, err, "failed to read status from database")
}

func TestManagerAlreadyInitError(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}
	passManagerProv := &PassManagerMock{}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.Init(context.Background(), "12")
	assert.EqualError(t, err, "vault is already initialized")
}

func TestManagerReadEncValueErr(t *testing.T) {
	dbProv := &DbProviderMock{}
	passManagerProv := &PassManagerMock{
		GetEncRandValueErr: errors.New("cannot enc pass"),
	}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.Init(context.Background(), "12")
	assert.EqualError(t, err, "cannot enc pass")
}

func TestManagerSetStatusErr(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToStoreErr: errors.New("cannot store status in db"),
	}
	passManagerProv := &PassManagerMock{}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.Init(context.Background(), "12")
	assert.EqualError(t, err, "cannot store status in db")
}

func TestUnlock(t *testing.T) {
	dbStatus := DbStatus{
		ID:            1,
		StatusName:    DbStatusInit,
		EncCheckValue: "123",
		DecCheckValue: "345",
	}
	dbProv := &DbProviderMock{
		statusToGive: dbStatus,
	}
	passManagerProv := &PassManagerMock{
		PassMatchToGive: true,
	}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	assert.True(t, mngr.IsLocked())

	err := mngr.UnLock(context.Background(), "12")
	require.NoError(t, err)
	assert.False(t, mngr.IsLocked())
	assert.Equal(t, dbStatus, passManagerProv.PassMatchDbStatusGiven)
}

func TestUnlockWhenAlreadyUnlocked(t *testing.T) {
	dbStatus := DbStatus{
		StatusName: DbStatusInit,
	}
	dbProv := &DbProviderMock{
		statusToGive: dbStatus,
	}
	passManagerProv := &PassManagerMock{
		PassMatchToGive: true,
	}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.UnLock(context.Background(), "12")
	require.NoError(t, err)

	err = mngr.UnLock(context.Background(), "12")
	require.EqualError(t, err, "vault is already unlocked")

	appErr, ok := err.(errors2.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusConflict, appErr.Code)
}

func TestUnlockStatusReadError(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGiveErr: errors.New("status read error"),
	}
	passManagerProv := &PassManagerMock{}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.UnLock(context.Background(), "12")
	require.EqualError(t, err, "status read error")
}

func TestUnlockWhenDbIsNotInit(t *testing.T) {
	dbProv := &DbProviderMock{}
	passManagerProv := &PassManagerMock{}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.UnLock(context.Background(), "12")
	require.EqualError(t, err, "vault is not yet initialized")

	appErr, ok := err.(errors2.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusConflict, appErr.Code)
}

func TestUnlockPasswordCheckError(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}
	passManagerProv := &PassManagerMock{
		PassMatchErr: errors.New("pass match error"),
	}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.UnLock(context.Background(), "12")
	require.EqualError(t, err, "pass match error")
}

func TestUnlockWithWrongPassword(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}
	passManagerProv := &PassManagerMock{
		PassMatchToGive: false,
	}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.UnLock(context.Background(), "12")
	require.EqualError(t, err, WrongPasswordError.Error())

	appErr, ok := err.(errors2.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, appErr.Code)
}

func TestLock(t *testing.T) {
	dbStatus := DbStatus{
		StatusName: DbStatusInit,
	}
	dbProv := &DbProviderMock{
		statusToGive: dbStatus,
	}
	passManagerProv := &PassManagerMock{
		PassMatchToGive: true,
	}
	mngr := NewManager(dbProv, passManagerProv, testLog)
	err := mngr.UnLock(context.Background(), "123")
	require.NoError(t, err)
	assert.False(t, mngr.IsLocked())

	err = mngr.Lock(context.Background())
	require.NoError(t, err)
	assert.True(t, mngr.IsLocked())
}

func TestLockWhenNotUnlocked(t *testing.T) {
	dbProv := &DbProviderMock{}
	passManagerProv := &PassManagerMock{}
	mngr := NewManager(dbProv, passManagerProv, testLog)

	err := mngr.Lock(context.Background())
	require.EqualError(t, err, "vault is already locked")

	appErr, ok := err.(errors2.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusConflict, appErr.Code)
}

func TestLockWithReadStatusError(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGiveErr: errors.New("failed to read db status"),
	}
	passManagerProv := &PassManagerMock{}

	mngr := NewManager(dbProv, passManagerProv, testLog)
	mngr.pass = "123"

	err := mngr.Lock(context.Background())
	require.EqualError(t, err, "failed to read db status")
}

func TestLockWhenDBIsNotInitialized(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: DbStatusNotInit,
		},
	}
	passManagerProv := &PassManagerMock{}

	mngr := NewManager(dbProv, passManagerProv, testLog)
	mngr.pass = "123"

	err := mngr.Lock(context.Background())
	require.EqualError(t, err, "vault is not yet initialized")
}

func TestReadStatusNotInitialised(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: DbStatusNotInit,
		},
	}
	passManagerProv := &PassManagerMock{}

	mngr := NewManager(dbProv, passManagerProv, testLog)
	st, err := mngr.Status(context.Background())
	require.NoError(t, err)

	assert.Equal(
		t,
		StatusReport{
			InitStatus: DbStatusNotInit,
			LockStatus: "",
		},
		st,
	)

	dbProv = &DbProviderMock{
		statusToGive: DbStatus{},
	}
	mngr = NewManager(dbProv, &PassManagerMock{}, testLog)
	st, err = mngr.Status(context.Background())
	require.NoError(t, err)

	assert.Equal(
		t,
		StatusReport{
			InitStatus: DbStatusNotInit,
			LockStatus: "",
		},
		st,
	)

	dbProv = &DbProviderMock{
		statusToGiveErr: ErrDatabaseNotInitialised,
	}
	mngr = NewManager(dbProv, &PassManagerMock{}, testLog)
	st, err = mngr.Status(context.Background())
	require.NoError(t, err)

	assert.Equal(
		t,
		StatusReport{
			InitStatus: DbStatusNotInit,
			LockStatus: "",
		},
		st,
	)
}

func TestReadStatusLocked(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}
	passManagerProv := &PassManagerMock{}

	mngr := NewManager(dbProv, passManagerProv, testLog)
	st, err := mngr.Status(context.Background())
	require.NoError(t, err)

	assert.Equal(
		t,
		StatusReport{
			InitStatus: DbStatusInit,
			LockStatus: StatusLocked,
		},
		st,
	)
}

func TestReadStatusUnLocked(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)
	mngr.pass = "123"
	st, err := mngr.Status(context.Background())
	require.NoError(t, err)

	assert.Equal(
		t,
		StatusReport{
			InitStatus: DbStatusInit,
			LockStatus: StatusUnlocked,
		},
		st,
	)
}

func TestReadStatusFailure(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGiveErr: errors.New("failed to read status"),
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)
	_, err := mngr.Status(context.Background())
	require.EqualError(t, err, "failed to read status")
}

func TestManagerList(t *testing.T) {
	expectedValueKeys := []ValueKey{
		{
			ID:        1,
			ClientID:  "client1",
			CreatedBy: "user1",
			CreatedAt: time.Now(),
			Key:       "key1",
		},
	}
	dbProv := &DbProviderMock{
		listValuesToGive: expectedValueKeys,
		statusToGive: DbStatus{
			StatusName: "",
		},
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)

	inputURL, err := url.Parse("/someu?sort=key&sort=-created_at&filter[client_id]=val1")
	require.NoError(t, err)

	req := &http.Request{
		URL: inputURL,
	}

	_, err = mngr.List(context.Background(), req)
	require.EqualError(t, err, "vault is locked")

	mngr.pass = "123"

	_, err = mngr.List(context.Background(), req)
	require.EqualError(t, err, "vault is not initialized")

	dbProv.statusToGive = DbStatus{
		StatusName: DbStatusInit,
	}

	actualValues, err := mngr.List(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(
		t,
		&ListOptions{
			Sorts: []SortOption{
				{
					Column: "key",
					IsASC:  true,
				},
				{
					Column: "created_at",
					IsASC:  false,
				},
			},
			Filters: []FilterOption{
				{
					Column: "client_id",
					Values: []string{"val1"},
				},
			},
		},
		dbProv.listOptionInput,
	)
	assert.Equal(t, expectedValueKeys, actualValues)

	dbProv = &DbProviderMock{
		listErrorToGive: errors.New("list error"),
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}

	mngr = NewManager(dbProv, &PassManagerMock{}, testLog)
	mngr.pass = "123"

	_, err = mngr.List(context.Background(), req)
	require.EqualError(t, err, "list error")
}

func TestListWithUnsupportedFilterAndSort(t *testing.T) {
	dbProv := &DbProviderMock{
		listValuesToGive: []ValueKey{},
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)
	mngr.pass = "123"

	inputURL, err := url.Parse("/someu?sort=unsupportedSortField&filter[unsupportedFilter]=val1")
	require.NoError(t, err)

	req := &http.Request{
		URL: inputURL,
	}

	_, err = mngr.List(context.Background(), req)
	require.EqualError(t, err, "unsupported sort field 'unsupportedSortField', unsupported filter field 'unsupportedFilter'")
}

func TestGetOne(t *testing.T) {
	const pass = "1234"
	encValue, err := enc.Aes256EncryptByPassToBase64String([]byte("some val"), pass)
	require.NoError(t, err)

	givenStoredValue := StoredValue{
		InputValue: InputValue{
			Value: encValue,
			Key:   "somekey1",
		},
		ID:        1,
		CreatedBy: "guy",
	}
	expectedValue := StoredValue{
		InputValue: InputValue{
			Value: "some val",
			Key:   "somekey1",
		},
		ID:        1,
		CreatedBy: "guy",
	}
	dbProv := &DbProviderMock{
		getByIDStoredValue: givenStoredValue,
		getByIDFound:       true,
		statusToGive: DbStatus{
			StatusName: "",
		},
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)

	user := &UserDataProviderMock{
		GroupsToGive: []string{},
	}

	_, _, err = mngr.GetOne(context.Background(), 1, user)
	require.EqualError(t, err, "vault is locked")

	mngr.pass = pass

	_, _, err = mngr.GetOne(context.Background(), 1, user)
	require.EqualError(t, err, "vault is not initialized")

	dbProv.statusToGive = DbStatus{
		StatusName: DbStatusInit,
	}

	val, found, err := mngr.GetOne(context.Background(), 1, user)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, expectedValue, val)

	dbProv = &DbProviderMock{
		getByIDStoredValue: givenStoredValue,
		getByIDFound:       false,
		getByIDError:       nil,
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}

	mngr = NewManager(dbProv, &PassManagerMock{}, testLog)
	mngr.pass = "123"

	_, found, err = mngr.GetOne(context.Background(), 1, user)
	require.NoError(t, err)
	assert.False(t, found)

	dbProv = &DbProviderMock{
		getByIDStoredValue: givenStoredValue,
		getByIDFound:       false,
		getByIDError:       errors.New("some get id error"),
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}

	mngr = NewManager(dbProv, &PassManagerMock{}, testLog)
	mngr.pass = pass

	_, _, err = mngr.GetOne(context.Background(), 1, user)
	require.EqualError(t, err, "some get id error")
}

func TestGetOneWithLimitedAccess(t *testing.T) {
	const pass = "1234"
	encValue, err := enc.Aes256EncryptByPassToBase64String([]byte("some val"), pass)
	require.NoError(t, err)

	givenStoredValue := StoredValue{
		InputValue: InputValue{
			RequiredGroup: "root",
			Value:         encValue,
		},
	}
	dbProv := &DbProviderMock{
		getByIDStoredValue: givenStoredValue,
		getByIDFound:       true,
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)
	mngr.pass = pass

	user := &UserDataProviderMock{
		GroupsToGive: []string{},
	}

	_, _, err = mngr.GetOne(context.Background(), 1, user)
	assert.Equal(
		t,
		errors2.APIError{
			Message: "your group doesn't allow access to this value",
			Code:    http.StatusForbidden,
		},
		err,
	)

	user2 := &UserDataProviderMock{
		GroupsToGive: []string{"root"},
	}

	_, _, err = mngr.GetOne(context.Background(), 1, user2)
	assert.NoError(t, err)
}

func TestStore(t *testing.T) {
	const pass = "1234"

	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: "",
		},
		SaveIDToGive: 123,
	}

	inputValue := &InputValue{
		ClientID:      "client1",
		RequiredGroup: "group1",
		Key:           "someKey",
		Value:         "someValue",
		Type:          SecreteType,
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)

	user := UserDataProviderMock{
		UsernameToGive: "someuser",
		GroupsToGive:   []string{},
	}
	t.Run("vault_locked", func(t *testing.T) {
		_, err := mngr.Store(context.Background(), 0, inputValue, user)
		require.EqualError(t, err, "vault is locked")
	})

	mngr.pass = pass

	t.Run("vault_not_init", func(t *testing.T) {
		_, err := mngr.Store(context.Background(), 1, inputValue, user)
		require.EqualError(t, err, "vault is not initialized")
	})

	dbProv.statusToGive = DbStatus{
		StatusName: DbStatusInit,
	}

	t.Run("create_success", func(t *testing.T) {
		storedID, err := mngr.Store(context.Background(), 0, inputValue, user)
		require.NoError(t, err)
		assert.Equal(t, int64(123), storedID.ID)

		assert.Equal(t, "someuser", dbProv.SaveUserGiven)
		assert.Equal(t, int64(0), dbProv.SaveIDGiven)
		assert.True(t, dbProv.SaveNowDateGiven.Equal(time.Now()) || dbProv.SaveNowDateGiven.Before(time.Now()))

		actualInputValue := dbProv.SaveInputGiven
		assert.Equal(t, "client1", actualInputValue.ClientID)
		assert.Equal(t, "group1", actualInputValue.RequiredGroup)
		assert.Equal(t, "someKey", actualInputValue.Key)
		assert.Equal(t, SecreteType, actualInputValue.Type)

		actualDecryptedValue, err := enc.Aes256DecryptByPassFromBase64String(actualInputValue.Value, pass)
		require.NoError(t, err)
		assert.Equal(t, "someValue", string(actualDecryptedValue))
	})

	dbProv.getByIDFound = true
	dbProv.getByIDStoredValue = StoredValue{
		InputValue: InputValue{},
	}
	t.Run("update_success", func(t *testing.T) {
		_, err := mngr.Store(context.Background(), 1, inputValue, user)
		require.NoError(t, err)
	})

	dbProv.FindByKeyAndClientIDFoundToGive = true
	t.Run("store_failure_key_exists", func(t *testing.T) {
		_, err := mngr.Store(context.Background(), 1, inputValue, user)
		require.EqualError(t, err, "another key 'someKey' exists for this client 'client1'")
	})

	dbProv.FindByKeyAndClientIDFoundToGive = false

	dbProv.FindByKeyAndClientIDErrorToGive = errors.New("finding key and client error")
	t.Run("store_failure_key_exists_error", func(t *testing.T) {
		_, err := mngr.Store(context.Background(), 1, inputValue, user)
		require.EqualError(t, err, "finding key and client error")
	})

	dbProv.FindByKeyAndClientIDErrorToGive = nil

	t.Run("invalid_input", func(t *testing.T) {
		_, err := mngr.Store(context.Background(), 1, &InputValue{}, user)
		require.EqualError(t, err, "key is required, value is required, value type is required")
	})

	dbProv.SaveErrorToGive = errors.New("failed to save value to db")
	t.Run("db_store_error", func(t *testing.T) {
		_, err := mngr.Store(context.Background(), 1, inputValue, user)
		require.EqualError(t, err, "failed to save value to db")
	})
}

func TestStoreWithLimitedGroupAccess(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
	}
	dbProv.getByIDFound = true
	dbProv.getByIDStoredValue = StoredValue{
		InputValue: InputValue{
			ClientID:      "client123",
			RequiredGroup: "secure_group",
			Key:           "key",
			Value:         "val",
			Type:          SecreteType,
		},
		ID:        1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "admin",
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)
	mngr.pass = "12345"

	user := UserDataProviderMock{
		UsernameToGive: "someuser",
		GroupsToGive:   []string{},
	}

	_, err := mngr.Store(context.Background(), 1, &dbProv.getByIDStoredValue.InputValue, user)
	require.Equal(
		t,
		errors2.APIError{
			Message: "your group doesn't allow access to this value",
			Code:    http.StatusForbidden,
		},
		err,
	)

	user2 := UserDataProviderMock{
		UsernameToGive: "someuser",
		GroupsToGive:   []string{"secure_group"},
	}

	_, err = mngr.Store(context.Background(), 1, &dbProv.getByIDStoredValue.InputValue, user2)
	assert.NoError(t, err)
}

func TestDeleteKey(t *testing.T) {
	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: "",
		},
		getByIDFound: true,
		getByIDStoredValue: StoredValue{
			InputValue: InputValue{},
		},
	}

	user := UserDataProviderMock{
		UsernameToGive: "someuser",
		GroupsToGive:   []string{},
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)

	t.Run("vault_locked", func(t *testing.T) {
		err := mngr.Delete(context.Background(), 1, user)
		require.EqualError(t, err, "vault is locked")
	})

	mngr.pass = "1234"

	t.Run("vault_not_init", func(t *testing.T) {
		err := mngr.Delete(context.Background(), 1, user)
		require.EqualError(t, err, "vault is not initialized")
	})

	dbProv.statusToGive.StatusName = DbStatusInit

	t.Run("delete_success", func(t *testing.T) {
		err := mngr.Delete(context.Background(), 1, user)
		require.NoError(t, err)

		assert.Equal(t, 1, dbProv.DeleteIDGiven)
	})

	dbProv.DeleteErrorToGive = errors.New("failed to delete value to db")
	t.Run("db_error", func(t *testing.T) {
		err := mngr.Delete(context.Background(), 1, user)
		require.EqualError(t, err, "failed to delete value to db")
	})
	dbProv.DeleteErrorToGive = nil

	dbProv.getByIDFound = false
	t.Run("entryNotFound", func(t *testing.T) {
		err := mngr.Delete(context.Background(), 1, user)
		require.Equal(
			t,
			errors2.APIError{
				Message: "cannot find this entry by the provided id",
				Code:    http.StatusNotFound,
			},
			err,
		)
	})

	dbProv.getByIDFound = true
	dbProv.getByIDError = errors.New("failed to read stored value")
	t.Run("entry read error", func(t *testing.T) {
		err := mngr.Delete(context.Background(), 1, user)
		require.EqualError(t, err, "failed to read stored value")
	})
}

func TestDeleteKeyWithNoGroupAccess(t *testing.T) {
	const pass = "1234"

	dbProv := &DbProviderMock{
		statusToGive: DbStatus{
			StatusName: DbStatusInit,
		},
		getByIDFound: true,
		getByIDStoredValue: StoredValue{
			InputValue: InputValue{
				RequiredGroup: "secure_group",
			},
		},
	}

	mngr := NewManager(dbProv, &PassManagerMock{}, testLog)

	mngr.pass = pass

	t.Run("user_no_key_access", func(t *testing.T) {
		user := UserDataProviderMock{
			UsernameToGive: "someuser",
			GroupsToGive:   []string{},
		}

		err := mngr.Delete(context.Background(), 1, user)
		require.Equal(
			t,
			errors2.APIError{
				Message: "your group doesn't allow access to this value",
				Code:    http.StatusForbidden,
			},
			err,
		)
	})

	t.Run("user_with_key_access", func(t *testing.T) {
		user2 := UserDataProviderMock{
			UsernameToGive: "someuser",
			GroupsToGive:   []string{"secure_group"},
		}

		err := mngr.Delete(context.Background(), 1, user2)
		require.NoError(t, err)
	})
}
