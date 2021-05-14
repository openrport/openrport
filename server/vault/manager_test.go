package vault

import (
	"context"
	"errors"
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
	SaveIDGiven      int
	SaveInputGiven   *InputValue
	SaveNowDateGiven time.Time
	SaveErrorToGive  error

	DeleteIDGiven     int
	DeleteErrorToGive error
}

func (dpm *DbProviderMock) Init(ctx context.Context) error {
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

func (dpm *DbProviderMock) Save(ctx context.Context, user string, idToUpdate int, val *InputValue, nowDate time.Time) error {
	dpm.SaveUserGiven = user
	dpm.SaveIDGiven = idToUpdate
	dpm.SaveInputGiven = val
	dpm.SaveNowDateGiven = nowDate

	return dpm.SaveErrorToGive
}

func (dpm *DbProviderMock) Delete(ctx context.Context, id int) error {
	dpm.DeleteIDGiven = id

	return dpm.DeleteErrorToGive
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

	inputURL, err := url.Parse("/someu?sort=date&sort=-user&filter[field1]=val1")
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
					Column: "date",
					IsASC:  true,
				},
				{
					Column: "user",
					IsASC:  false,
				},
			},
			Filters: []FilterOption{
				{
					Column: "field1",
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

	_, _, err = mngr.GetOne(context.Background(), 1)
	require.EqualError(t, err, "vault is locked")

	mngr.pass = pass

	_, _, err = mngr.GetOne(context.Background(), 1)
	require.EqualError(t, err, "vault is not initialized")

	dbProv.statusToGive = DbStatus{
		StatusName: DbStatusInit,
	}

	val, found, err := mngr.GetOne(context.Background(), 1)
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

	_, found, err = mngr.GetOne(context.Background(), 1)
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

	_, _, err = mngr.GetOne(context.Background(), 1)
	require.EqualError(t, err, "some get id error")
}
