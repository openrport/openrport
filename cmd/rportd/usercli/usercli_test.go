package usercli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openrport/openrport/share/enums"
)

func TestCreateUser(t *testing.T) {
	mockUserService := &userServiceMock{}
	mockPasswordReader := &passwordReaderMock{
		Passwords: []string{"testpassword", "testpassword"},
	}

	err := CreateUser(mockUserService, mockPasswordReader, "testuser", []string{"group1", "group2"}, "test-2fa")
	require.NoError(t, err)

	expected := UserInput{
		Username:    "testuser",
		Password:    "testpassword",
		Groups:      []string{"group1", "group2"},
		TwoFASendTo: "test-2fa",
	}
	assert.True(t, mockUserService.CreateCalled)
	assert.Equal(t, expected, mockUserService.UserInput)
	assert.Equal(t, 2, mockPasswordReader.CallCount)
}

func TestUpdateUser(t *testing.T) {
	testCases := []struct {
		Name           string
		AskForPassword bool
		Groups         []string
		TwoFASendTo    string
		Expected       UserInput
	}{
		{
			Name:           "full",
			AskForPassword: true,
			Groups:         []string{"group1", "group2"},
			TwoFASendTo:    "test-2fa",
			Expected: UserInput{
				Username:    "testuser",
				Password:    "testpassword",
				Groups:      []string{"group1", "group2"},
				TwoFASendTo: "test-2fa",
			},
		},
		{
			Name:           "groups only",
			AskForPassword: false,
			Groups:         []string{"group1", "group2"},
			Expected: UserInput{
				Username: "testuser",
				Groups:   []string{"group1", "group2"},
			},
		},
		{
			Name:           "password only",
			AskForPassword: true,
			Expected: UserInput{
				Username: "testuser",
				Password: "testpassword",
			},
		},
	}

	for _, tc := range testCases {
		mockUserService := &userServiceMock{}
		mockPasswordReader := &passwordReaderMock{
			Passwords: []string{"testpassword", "testpassword"},
		}

		err := UpdateUser(mockUserService, mockPasswordReader, "testuser", tc.Groups, tc.TwoFASendTo, tc.AskForPassword)
		require.NoError(t, err)

		assert.True(t, mockUserService.ChangeCalled)

		assert.Equal(t, tc.Expected, mockUserService.UserInput)
		if tc.AskForPassword {
			assert.Equal(t, 2, mockPasswordReader.CallCount)
		} else {
			assert.Equal(t, 0, mockPasswordReader.CallCount)
		}
	}
}

func TestDeleteUser(t *testing.T) {
	mockUserService := &userServiceMock{}

	err := DeleteUser(mockUserService, "testuser")
	require.NoError(t, err)

	assert.Equal(t, "testuser", mockUserService.DeletedUser)
}

type userServiceMock struct {
	CreateCalled bool
	ChangeCalled bool
	UserInput    UserInput
	DeletedUser  string
}

func (m *userServiceMock) Create(input UserInput) error {
	m.CreateCalled = true
	m.UserInput = input
	return nil
}

func (m *userServiceMock) Change(input UserInput) error {
	m.ChangeCalled = true
	m.UserInput = input
	return nil
}

func (m *userServiceMock) Delete(username string) error {
	m.DeletedUser = username
	return nil
}

func (m *userServiceMock) ProviderType() enums.ProviderSource {
	return enums.ProviderSourceMock
}
