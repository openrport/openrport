package users

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share/logger"
)

var user1 = &User{
	Username: "user1",
	Password: "pass0123456789",
	Groups:   []string{"group1", "group2"},
	TotP:     "totp123",
}

var user2 = &User{
	Username: "user2",
	Password: "pass2123456789",
	Groups:   []string{"group2", "group3"},
}

type FileManagerMock struct {
	Users []*User
}

func (fmm *FileManagerMock) ReadUsersFromFile() ([]*User, error) {
	return fmm.Users, nil
}

func (fmm *FileManagerMock) SaveUsersToFile(users []*User) error {
	fmm.Users = users
	return nil
}

func TestFileAdapterLoad(t *testing.T) {
	logger := chshare.NewLogger("file-adapter-test", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)
	fileManagerMock := &FileManagerMock{
		Users: []*User{user1},
	}
	fa, err := NewFileAdapter(logger, fileManagerMock)
	require.NoError(t, err)

	users, err := fa.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*User{user1}, users)

	fileManagerMock.Users = []*User{user1, user2}
	err = fa.load()
	require.NoError(t, err)

	users, err = fa.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*User{user1, user2}, users)
}

func TestFileAdapterAdd(t *testing.T) {
	logger := chshare.NewLogger("file-adapter-test", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)
	fileManagerMock := &FileManagerMock{
		Users: []*User{user1},
	}
	fa, err := NewFileAdapter(logger, fileManagerMock)
	require.NoError(t, err)

	// add new user
	err = fa.Add(user2)
	require.NoError(t, err)

	users, err := fa.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*User{user1, user2}, users)

	// error when adding existing user
	err = fa.Add(user1)

	assert.Error(t, err)
}

func TestFileAdapterDelete(t *testing.T) {
	logger := chshare.NewLogger("file-adapter-test", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)
	fileManagerMock := &FileManagerMock{
		Users: []*User{user1},
	}
	fa, err := NewFileAdapter(logger, fileManagerMock)
	require.NoError(t, err)

	// delete existing user
	err = fa.Delete(user1.Username)
	require.NoError(t, err)

	users, err := fa.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*User{}, users)

	// error when deleting non-existent user
	err = fa.Delete(user2.Username)

	assert.Error(t, err)
}

func TestFileAdapterUpdate(t *testing.T) {
	logger := chshare.NewLogger("file-adapter-test", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)
	fileManagerMock := &FileManagerMock{
		Users: []*User{user1, user2},
	}
	fa, err := NewFileAdapter(logger, fileManagerMock)
	require.NoError(t, err)

	updates := &User{
		Username: "user3",
		Password: "pass3",
		Token:    Token("token3"),
		Groups:   []string{"group1", "group4"},
		TotP:     "totp123",
	}

	// update existing user
	err = fa.Update(updates, user1.Username)
	require.NoError(t, err)

	users, err := fa.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*User{updates, user2}, users)

	// error when updating non-existent user
	err = fa.Update(updates, "non-existent")

	assert.Error(t, err)

	// error when updating to username that exists
	updates.Username = "user2"
	err = fa.Update(updates, user1.Username)

	assert.Error(t, err)
}
