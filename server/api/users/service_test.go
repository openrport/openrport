package users

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/share/enums"
)

type ProviderMock struct {
	UsersToGive         []*User
	UsersToAdd          []*User
	UsersToUpdate       []*User
	GroupsToGive        []Group
	ErrorToGiveOnRead   error
	ErrorToGiveOnWrite  error
	ErrorToGiveOnDelete error
	UsernameToUpdate    string
	UsernameToDelete    string
}

func (dpm *ProviderMock) GetAll() ([]*User, error) {
	return dpm.UsersToGive, dpm.ErrorToGiveOnRead
}

func (dpm *ProviderMock) ListGroups() ([]Group, error) {
	return dpm.GroupsToGive, dpm.ErrorToGiveOnRead
}

func (dpm *ProviderMock) GetGroup(name string) (Group, error) {
	for _, g := range dpm.GroupsToGive {
		if g.Name == name {
			return g, dpm.ErrorToGiveOnRead
		}
	}
	return NewGroup(name), dpm.ErrorToGiveOnRead
}

func (dpm *ProviderMock) UpdateGroup(string, Group) error {
	return dpm.ErrorToGiveOnWrite
}

func (dpm *ProviderMock) DeleteGroup(string) error {
	return dpm.ErrorToGiveOnDelete
}

func (dpm *ProviderMock) GetByUsername(username string) (*User, error) {
	var usr *User
	for i := range dpm.UsersToGive {
		if dpm.UsersToGive[i].Username == username {
			usr = dpm.UsersToGive[i]
		}
	}

	return usr, dpm.ErrorToGiveOnRead
}

func (dpm *ProviderMock) Add(usr *User) error {
	if dpm.UsersToAdd == nil {
		dpm.UsersToAdd = []*User{}
	}

	dpm.UsersToAdd = append(dpm.UsersToAdd, usr)

	return dpm.ErrorToGiveOnWrite
}

func (dpm *ProviderMock) Update(usr *User, usernameToUpdate string) error {
	if dpm.UsersToUpdate == nil {
		dpm.UsersToUpdate = []*User{}
	}

	dpm.UsersToUpdate = append(dpm.UsersToUpdate, usr)
	dpm.UsernameToUpdate = usernameToUpdate

	return dpm.ErrorToGiveOnWrite
}

func (dpm *ProviderMock) Delete(usernameToDelete string) error {
	dpm.UsernameToDelete = usernameToDelete
	return dpm.ErrorToGiveOnDelete
}

func (dpm *ProviderMock) SupportsGroupPermissions() bool {
	return false
}

func (dpm ProviderMock) Type() enums.ProviderSource {
	return enums.ProviderSourceDB
}

func TestGetUsersFromProvider(t *testing.T) {
	givenUsers := []*User{
		{
			Username: "one",
			Password: "two",
			Groups:   []string{"group1"},
		},
	}
	db := &ProviderMock{
		UsersToGive: givenUsers,
	}

	service := APIService{
		Provider: db,
	}

	actualUsers, err := service.GetAll()

	require.NoError(t, err)
	assert.Equal(t, givenUsers, actualUsers)

	db = &ProviderMock{
		UsersToGive:       givenUsers,
		ErrorToGiveOnRead: errors.New("some db error"),
	}

	service = APIService{
		Provider: db,
	}

	_, err = service.GetAll()
	require.EqualError(t, err, "some db error")
}

func TestValidate(t *testing.T) {
	service := APIService{}

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
				Username: "someuser ",
				Password: "123",
			},
			expectedError: "username must not start or end with whitespace",
			name:          "username ending with space",
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

func TestAddUser(t *testing.T) {
	givenUser := &User{
		Username: "user13",
		Password: "pass13",
	}

	dbProvider := &ProviderMock{}
	service := APIService{
		Provider: dbProvider,
	}

	err := service.Change(givenUser, "")
	require.NoError(t, err)

	require.Len(t, dbProvider.UsersToAdd, 1)
	assert.Equal(t, "user13", dbProvider.UsersToAdd[0].Username)
	assert.True(t, strings.HasPrefix(dbProvider.UsersToAdd[0].Password, HtpasswdBcryptPrefix))
	require.Len(t, dbProvider.UsersToUpdate, 0)

	dbProvider = &ProviderMock{
		ErrorToGiveOnRead: errors.New("some read error"),
	}
	service = APIService{
		Provider: dbProvider,
	}
	err = service.Change(givenUser, "")
	require.EqualError(t, err, "some read error")

	dbProvider = &ProviderMock{
		ErrorToGiveOnWrite: errors.New("some write error"),
	}
	service = APIService{
		Provider: dbProvider,
	}
	err = service.Change(givenUser, "")
	require.EqualError(t, err, "some write error")
}

func TestAddUserIfItExists(t *testing.T) {
	userToUpdate := &User{
		Username: "user1",
		Password: "pass1",
	}

	dbProvider := &ProviderMock{
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
		Provider: dbProvider,
	}

	err := service.Change(userToUpdate, "")
	require.EqualError(t, err, "Another user with this username already exists")
	require.Len(t, dbProvider.UsersToAdd, 0)
	require.Len(t, dbProvider.UsersToUpdate, 0)
}

func TestUpdateUserIfItExists(t *testing.T) {
	givenUser := &User{
		Username: "user1",
		Password: "pass1",
	}

	dbProvider := &ProviderMock{
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
		Provider: dbProvider,
	}

	err := service.Change(givenUser, "user2")
	require.EqualError(t, err, "Another user with this username already exists")
	require.Len(t, dbProvider.UsersToAdd, 0)
	require.Len(t, dbProvider.UsersToUpdate, 0)
}

func TestUpdateUserInProvider(t *testing.T) {
	userToUpdate := &User{
		Username: "user_one",
		Password: "pass_one",
		Groups:   []string{"group_one", "group_two"},
	}

	dbProvider := &ProviderMock{
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
		Provider: dbProvider,
	}

	err := service.Change(userToUpdate, "user1")
	require.NoError(t, err)

	assert.Len(t, dbProvider.UsersToAdd, 0)
	require.Len(t, dbProvider.UsersToUpdate, 1)
	assert.Equal(t, "user_one", dbProvider.UsersToUpdate[0].Username)
	assert.True(t, strings.HasPrefix(dbProvider.UsersToUpdate[0].Password, HtpasswdBcryptPrefix))
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
			Message:    "cannot find user by username 'non-existing-user'",
			HTTPStatus: http.StatusNotFound,
		},
		err,
	)

	service = APIService{
		Provider: &ProviderMock{
			ErrorToGiveOnWrite: errors.New("failed to write"),
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
	require.EqualError(t, err, "failed to write")
}

func TestDeleteUserFromProvider(t *testing.T) {
	dbProvider := &ProviderMock{
		UsersToGive: []*User{
			{
				Username: "user2",
			},
		},
	}

	service := APIService{
		Provider: dbProvider,
	}

	err := service.Delete("user2")
	require.NoError(t, err)
	assert.Equal(t, "user2", dbProvider.UsernameToDelete)

	err = service.Delete("unknown_user")
	assert.Equal(
		t,
		errors2.APIError{
			Message:    "cannot find user by username 'unknown_user'",
			HTTPStatus: http.StatusNotFound,
		},
		err,
	)

	dbProvider = &ProviderMock{
		UsersToGive: []*User{
			{
				Username: "user2",
			},
		},
		ErrorToGiveOnDelete: errors.New("failed to delete"),
	}

	service = APIService{
		Provider: dbProvider,
	}

	err = service.Delete("user2")
	require.EqualError(t, err, "failed to delete")
}

func TestExistsUserGroups(t *testing.T) {
	givenGroups := []Group{NewGroup("group1"), NewGroup("group2"), NewGroup("group3"), NewGroup("group4"), AdministratorsGroup}
	db := &ProviderMock{
		GroupsToGive: givenGroups,
	}

	service := APIService{
		Provider: db,
	}

	gotErr1 := service.ExistGroups([]string{"group4"})
	require.NoError(t, gotErr1)

	gotErr2 := service.ExistGroups([]string{"group1", "group2", "group3", "group4", Administrators})
	require.NoError(t, gotErr2)

	gotErr3 := service.ExistGroups([]string{"group1", "group2", "admin", Administrators})
	require.EqualError(t, gotErr3, "user groups not found: admin")

	gotErr4 := service.ExistGroups([]string{"group1", "group2", "group3", "admin", Administrators, "group5"})
	require.EqualError(t, gotErr4, "user groups not found: admin, group5")

	db = &ProviderMock{
		GroupsToGive:      givenGroups,
		ErrorToGiveOnRead: errors.New("some error"),
	}

	service = APIService{
		Provider: db,
	}

	gotErr5 := service.ExistGroups([]string{"group1"})
	require.EqualError(t, gotErr5, "some error")
}

func TestCheckPermission(t *testing.T) {
	testCases := []struct {
		Name       string
		Permission string
		User       *User
		Expected   bool
	}{
		{
			Name:       "no groups",
			Permission: PermissionCommands,
			User:       &User{},
			Expected:   false,
		},
		{
			Name:       "no permissions",
			Permission: PermissionCommands,
			User: &User{
				Groups: []string{"group-no-permissions"},
			},
			Expected: false,
		},
		{
			Name:       "admin group",
			Permission: PermissionCommands,
			User: &User{
				Groups: []string{"group-no-permissions", Administrators},
			},
			Expected: true,
		},
		{
			Name:       "has permission",
			Permission: PermissionCommands,
			User: &User{
				Groups: []string{"group-commands", "group-no-permissions"},
			},
			Expected: true,
		},
		{
			Name:       "has other permission",
			Permission: PermissionScheduler,
			User: &User{
				Groups: []string{"group-commands", "group-no-permissions"},
			},
			Expected: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			givenGroups := []Group{NewGroup("group-no-permissions"), NewGroup("group-commands", PermissionCommands), AdministratorsGroup}
			db := &ProviderMock{
				GroupsToGive: givenGroups,
			}
			service := APIService{
				Provider: db,
			}

			err := service.CheckPermission(tc.User, tc.Permission)
			if tc.Expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
