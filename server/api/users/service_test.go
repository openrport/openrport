package users

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/share/enums"
)

type DBProviderMock struct {
	UsersToGive         []*User
	UsersToAdd          []*User
	UsersToUpdate       []*User
	ErrorToGiveOnRead   error
	ErrorToGiveOnWrite  error
	ErrorToGiveOnDelete error
	UsernameToUpdate    string
	UsernameToDelete    string
}

func (dpm *DBProviderMock) GetAll() ([]*User, error) {
	return dpm.UsersToGive, dpm.ErrorToGiveOnRead
}

func (dpm *DBProviderMock) GetByUsername(username string) (*User, error) {
	var usr *User
	for i := range dpm.UsersToGive {
		if dpm.UsersToGive[i].Username == username {
			usr = dpm.UsersToGive[i]
		}
	}

	return usr, dpm.ErrorToGiveOnRead
}

func (dpm *DBProviderMock) Add(usr *User) error {
	if dpm.UsersToAdd == nil {
		dpm.UsersToAdd = []*User{}
	}

	dpm.UsersToAdd = append(dpm.UsersToAdd, usr)

	return dpm.ErrorToGiveOnWrite
}

func (dpm *DBProviderMock) Update(usr *User, usernameToUpdate string) error {
	if dpm.UsersToUpdate == nil {
		dpm.UsersToUpdate = []*User{}
	}

	dpm.UsersToUpdate = append(dpm.UsersToUpdate, usr)
	dpm.UsernameToUpdate = usernameToUpdate

	return dpm.ErrorToGiveOnWrite
}

func (dpm *DBProviderMock) Delete(usernameToDelete string) error {
	dpm.UsernameToDelete = usernameToDelete
	return dpm.ErrorToGiveOnDelete
}

type FileManagerMock struct {
	UsersToRead        []*User
	WrittenUsers       []*User
	ErrorToGiveOnRead  error
	ErrorToGiveOnWrite error
}

func (fmm *FileManagerMock) ReadUsersFromFile() ([]*User, error) {
	return fmm.UsersToRead, fmm.ErrorToGiveOnRead
}

func (fmm *FileManagerMock) SaveUsersToFile(users []*User) error {
	fmm.WrittenUsers = users
	return fmm.ErrorToGiveOnWrite
}

func TestGetUsersFromDB(t *testing.T) {
	givenUsers := []*User{
		{
			Username: "one",
			Password: "two",
			Groups:   []string{"group1"},
		},
	}
	db := &DBProviderMock{
		UsersToGive: givenUsers,
	}

	service := APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           db,
	}

	actualUsers, err := service.GetAll()

	require.NoError(t, err)
	assert.Equal(t, givenUsers, actualUsers)

	db = &DBProviderMock{
		UsersToGive:       givenUsers,
		ErrorToGiveOnRead: errors.New("some db error"),
	}

	service = APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           db,
	}

	_, err = service.GetAll()
	require.EqualError(t, err, "some db error")
}

func TestGetUsersFromFile(t *testing.T) {
	givenUsers := []*User{
		{
			Username: "user1",
			Password: "pass1",
			Groups:   []string{"group1", "group2"},
		},
	}

	service := APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: &FileManagerMock{
			UsersToRead: givenUsers,
		},
	}

	actualUsers, err := service.GetAll()

	require.NoError(t, err)
	assert.Equal(t, givenUsers, actualUsers)

	service = APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: &FileManagerMock{
			ErrorToGiveOnRead: errors.New("some file error"),
		},
	}

	_, err = service.GetAll()
	require.EqualError(t, err, "some file error")
}

func TestAddUserToFile(t *testing.T) {
	givenUser := &User{
		Username: "user1",
		Password: "pass1",
		Groups:   []string{"group1", "group2"},
	}

	usersFileManager := &FileManagerMock{}
	service := APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: usersFileManager,
	}

	err := service.Change(givenUser, "")
	require.NoError(t, err)

	require.Len(t, usersFileManager.WrittenUsers, 1)
	assert.Equal(t, givenUser, usersFileManager.WrittenUsers[0])

	usersFileManager = &FileManagerMock{
		ErrorToGiveOnRead: errors.New("some read error"),
	}
	service = APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: usersFileManager,
	}
	err = service.Change(givenUser, "")
	require.EqualError(t, err, "some read error")

	usersFileManager = &FileManagerMock{
		ErrorToGiveOnWrite: errors.New("some write error"),
	}
	service = APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: usersFileManager,
	}
	err = service.Change(givenUser, "")
	require.EqualError(t, err, "some write error")
}

func TestAddUserIfItExists(t *testing.T) {
	givenUser := &User{
		Username: "user1",
		Password: "pass1",
	}

	usersFileManager := &FileManagerMock{
		UsersToRead: []*User{
			{
				Username: "user1",
				Password: "pass1",
			},
			{
				Username: "user2",
				Password: "pass2",
			},
		},
	}
	service := APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: usersFileManager,
	}

	err := service.Change(givenUser, "")
	require.EqualError(t, err, "Another user with this username already exists")
	require.Len(t, usersFileManager.WrittenUsers, 0)
}

func TestUnsupportedUserProvider(t *testing.T) {
	service := APIService{
		ProviderType: enums.ProviderSourceStatic,
	}

	_, err := service.GetAll()
	require.EqualError(t, err, fmt.Sprintf("unsupported user data provider type: %s", enums.ProviderSourceStatic))

	userToUpdate := &User{
		Username: "user_one",
		Password: "pass_one",
	}
	err = service.Change(userToUpdate, "")
	require.EqualError(t, err, fmt.Sprintf("unsupported user data provider type: %s", enums.ProviderSourceStatic))

	err = service.Delete("some")
	require.EqualError(t, err, fmt.Sprintf("unsupported user data provider type: %s", enums.ProviderSourceStatic))
}

func TestValidate(t *testing.T) {
	service := APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: &FileManagerMock{
			UsersToRead: []*User{},
		},
	}

	emailSrv, err := message.NewSMTPService("host:port", "", "", "", false)
	require.NoError(t, err)

	testCases := []struct {
		name             string
		expectedError    string
		userKeyToProvide string
		user             *User
		twoFAOn          bool
		deliverySrv      message.Service
	}{
		{
			user:             &User{},
			expectedError:    "nothing to change",
			name:             "empty user on update",
			userKeyToProvide: "some",
		},
		{
			user:          &User{},
			expectedError: "username is required, password is required",
			name:          "empty user on create",
		},
		{
			user: &User{
				Password: "123",
			},
			expectedError: "username is required",
			name:          "no username provided",
		},
		{
			user: &User{
				Username: "someuser",
			},
			expectedError: "password is required",
			name:          "no password provided",
		},
		{
			user: &User{
				Username: "user123",
			},
			expectedError:    "nothing to change",
			name:             "nothing to change for the same username",
			userKeyToProvide: "user123",
		},
		{
			twoFAOn: true,
			user: &User{
				Username: "user123",
				Password: "123",
			},
			expectedError: "two_fa_send_to is required",
			name:          "no two_fa_send_to provided on create when 2fa is enabled",
		},
		{
			twoFAOn: false,
			user: &User{
				Username:    "user123",
				TwoFASendTo: "some-receiver",
			},
			userKeyToProvide: "user123",
			expectedError:    "nothing to change",
			name:             "only two_fa_send_to is provided but 2fa is disabled",
		},
		{
			twoFAOn:     true,
			deliverySrv: emailSrv,
			user: &User{
				Username:    "user123",
				Password:    "123",
				TwoFASendTo: "invalid-email-format",
			},
			expectedError: "invalid two_fa_send_to: invalid email format",
			name:          "invalid two_fa_send_to email is provided when 2fa is enabled on add user",
		},
		{
			twoFAOn: true,
			deliverySrv: &message.ServiceMock{
				ReturnError: errors.New("fake error"),
			},
			user: &User{
				TwoFASendTo: "invalid-receiver",
			},
			userKeyToProvide: "user123",

			expectedError: "invalid two_fa_send_to: fake error",
			name:          "invalid two_fa_send_to is provided when 2fa is enabled on update user",
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		service.DeliverySrv = testCase.deliverySrv
		t.Run(testCase.name, func(t *testing.T) {
			service.TwoFAOn = testCase.twoFAOn
			err := service.Change(testCase.user, testCase.userKeyToProvide)
			require.EqualError(t, err, testCase.expectedError)
		})
	}
}

func TestUpdateUserInFile(t *testing.T) {
	userToUpdate := &User{
		Username: "user_one",
		Password: "pass_one",
		Groups:   []string{"group_one", "group_two"},
	}

	usersFileManager := &FileManagerMock{
		UsersToRead: []*User{
			{
				Username: "user2",
				Groups:   []string{"group1"},
			},
			{
				Username: "user1",
				Password: "pass1",
				Groups:   []string{"group1", "group2"},
			},
		},
	}
	service := APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: usersFileManager,
	}

	err := service.Change(userToUpdate, "user1")
	require.NoError(t, err)

	require.Len(t, usersFileManager.WrittenUsers, 2)
	assert.Equal(t, "user2", usersFileManager.WrittenUsers[0].Username)
	assert.Equal(t, []string{"group1"}, usersFileManager.WrittenUsers[0].Groups)
	assert.Equal(t, "user_one", usersFileManager.WrittenUsers[1].Username)
	assert.True(t, strings.HasPrefix(usersFileManager.WrittenUsers[1].Password, htpasswdBcryptPrefix))
	assert.Equal(t, []string{"group_one", "group_two"}, usersFileManager.WrittenUsers[1].Groups)

	userToUpdate2 := &User{
		Groups: []string{},
	}
	err = service.Change(userToUpdate2, "user2")
	require.NoError(t, err)
	assert.Equal(t, []string{}, usersFileManager.WrittenUsers[0].Groups)

	userToUpdate = &User{
		Username: "unknown_user",
		Password: "222",
	}
	err = service.Change(userToUpdate, "unknown_user")
	assert.Equal(
		t,
		errors2.APIError{
			Message: "cannot find user by username 'unknown_user'",
			Code:    http.StatusNotFound,
		},
		err,
	)

	service = APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: &FileManagerMock{
			ErrorToGiveOnWrite: errors.New("failed to write to file"),
			UsersToRead: []*User{
				{
					Username: "user2",
				},
			},
		},
	}

	userToUpdate = &User{
		Username: "user2",
		Password: "3342",
	}
	err = service.Change(userToUpdate, "user2")
	require.EqualError(t, err, "failed to write to file")
}

func TestAddUserToDB(t *testing.T) {
	givenUser := &User{
		Username: "user13",
		Password: "pass13",
	}

	dbProvider := &DBProviderMock{}
	service := APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           dbProvider,
	}

	err := service.Change(givenUser, "")
	require.NoError(t, err)

	require.Len(t, dbProvider.UsersToAdd, 1)
	assert.Equal(t, "user13", dbProvider.UsersToAdd[0].Username)
	assert.True(t, strings.HasPrefix(dbProvider.UsersToAdd[0].Password, htpasswdBcryptPrefix))
	require.Len(t, dbProvider.UsersToUpdate, 0)

	dbProvider = &DBProviderMock{
		ErrorToGiveOnRead: errors.New("some read error"),
	}
	service = APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           dbProvider,
	}
	err = service.Change(givenUser, "")
	require.EqualError(t, err, "some read error")

	dbProvider = &DBProviderMock{
		ErrorToGiveOnWrite: errors.New("some write error"),
	}
	service = APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           dbProvider,
	}
	err = service.Change(givenUser, "")
	require.EqualError(t, err, "some write error")
}

func TestAddUserToDBIfItExists(t *testing.T) {
	userToUpdate := &User{
		Username: "user1",
		Password: "pass1",
	}

	dbProvider := &DBProviderMock{
		UsersToGive: []*User{
			{
				Username: "user1",
				Password: "pass1",
			},
			{
				Username: "user2",
				Password: "pass2",
			},
		},
	}

	service := APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           dbProvider,
	}

	err := service.Change(userToUpdate, "")
	require.EqualError(t, err, "Another user with this username already exists")
	require.Len(t, dbProvider.UsersToAdd, 0)
	require.Len(t, dbProvider.UsersToUpdate, 0)
}

func TestUpdateUserToDBIfItExists(t *testing.T) {
	givenUser := &User{
		Username: "user1",
		Password: "pass1",
	}

	dbProvider := &DBProviderMock{
		UsersToGive: []*User{
			{
				Username: "user1",
				Password: "pass1",
			},
			{
				Username: "user2",
				Password: "pass2",
			},
		},
	}

	service := APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           dbProvider,
	}

	err := service.Change(givenUser, "user2")
	require.EqualError(t, err, "Another user with this username already exists")
	require.Len(t, dbProvider.UsersToAdd, 0)
	require.Len(t, dbProvider.UsersToUpdate, 0)
}

func TestUpdateUserInDB(t *testing.T) {
	userToUpdate := &User{
		Username: "user_one",
		Password: "pass_one",
		Groups:   []string{"group_one", "group_two"},
	}

	dbProvider := &DBProviderMock{
		UsersToGive: []*User{
			{
				Username: "user2",
			},
			{
				Username: "user1",
				Password: "pass1",
				Groups:   []string{"group1", "group2"},
			},
		},
	}
	service := APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           dbProvider,
	}

	err := service.Change(userToUpdate, "user1")
	require.NoError(t, err)

	assert.Len(t, dbProvider.UsersToAdd, 0)
	require.Len(t, dbProvider.UsersToUpdate, 1)
	assert.Equal(t, "user_one", dbProvider.UsersToUpdate[0].Username)
	assert.True(t, strings.HasPrefix(dbProvider.UsersToUpdate[0].Password, htpasswdBcryptPrefix))
	assert.Equal(t, []string{"group_one", "group_two"}, dbProvider.UsersToUpdate[0].Groups)

	dbProvider.UsersToUpdate = []*User{}
	userToUpdate2 := &User{Groups: []string{}}
	err = service.Change(userToUpdate2, "user1")
	require.NoError(t, err)
	assert.Equal(t, []string{}, dbProvider.UsersToUpdate[0].Groups)

	err = service.Change(userToUpdate, "non-existing-user")
	assert.Equal(
		t,
		errors2.APIError{
			Message: "cannot find user by username 'non-existing-user'",
			Code:    http.StatusNotFound,
		},
		err,
	)

	service = APIService{
		ProviderType: enums.ProviderSourceDB,
		DB: &DBProviderMock{
			ErrorToGiveOnWrite: errors.New("failed to write to DB"),
			UsersToGive: []*User{
				{
					Username: "user2",
				},
			},
		},
	}

	userToUpdate = &User{
		Username: "user2",
		Password: "3342",
	}
	err = service.Change(userToUpdate, "user2")
	require.EqualError(t, err, "failed to write to DB")
}

func TestDeleteUserFromDB(t *testing.T) {
	dbProvider := &DBProviderMock{
		UsersToGive: []*User{
			{
				Username: "user2",
			},
		},
	}

	service := APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           dbProvider,
	}

	err := service.Delete("user2")
	require.NoError(t, err)
	assert.Equal(t, "user2", dbProvider.UsernameToDelete)

	err = service.Delete("unknown_user")
	assert.Equal(
		t,
		errors2.APIError{
			Message: "cannot find user by username 'unknown_user'",
			Code:    http.StatusNotFound,
		},
		err,
	)

	dbProvider = &DBProviderMock{
		UsersToGive: []*User{
			{
				Username: "user2",
			},
		},
		ErrorToGiveOnDelete: errors.New("failed to delete from db"),
	}

	service = APIService{
		ProviderType: enums.ProviderSourceDB,
		DB:           dbProvider,
	}

	err = service.Delete("user2")
	require.EqualError(t, err, "failed to delete from db")
}

func TestDeleteUserFromFile(t *testing.T) {
	usersFileManager := &FileManagerMock{
		UsersToRead: []*User{
			{
				Username: "user2",
			},
			{
				Username: "user1",
				Password: "pass1",
				Groups:   []string{"group1", "group2"},
			},
		},
	}

	service := APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: usersFileManager,
	}

	err := service.Delete("user2")
	require.NoError(t, err)
	expectedUsers := []*User{
		{
			Username: "user1",
			Password: "pass1",
			Groups:   []string{"group1", "group2"},
		},
	}
	assert.Equal(t, expectedUsers, usersFileManager.WrittenUsers)

	err = service.Delete("unknown_user")
	assert.Equal(
		t,
		errors2.APIError{
			Message: "cannot find user by username 'unknown_user'",
			Code:    http.StatusNotFound,
		},
		err,
	)

	usersFileManager = &FileManagerMock{
		ErrorToGiveOnRead: errors.New("failed to read users from file"),
	}
	service = APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: usersFileManager,
	}
	err = service.Delete("user2")
	require.EqualError(t, err, "failed to read users from file")

	usersFileManager = &FileManagerMock{
		UsersToRead: []*User{
			{
				Username: "user3",
			},
		},
		ErrorToGiveOnWrite: errors.New("failed to write users to file"),
	}
	service = APIService{
		ProviderType: enums.ProviderSourceFile,
		FileProvider: usersFileManager,
	}
	err = service.Delete("user3")
	require.EqualError(t, err, "failed to write users to file")
}
