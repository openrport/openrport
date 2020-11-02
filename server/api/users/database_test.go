package users

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserDatabase(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("CREATE TABLE `users` (username TEXT PRIMARY KEY, password TEXT)")
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE `groups` (username TEXT, `group` TEXT)")
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE `invalid_users` (username TEXT PRIMARY KEY, pass TEXT)")
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE `invalid_groups` (username TEXT, other TEXT)")
	require.NoError(t, err)

	testCases := []struct {
		Name          string
		UsersTable    string
		GroupsTable   string
		ExpectedError string
	}{
		{
			Name:          "invalid users tables",
			UsersTable:    "non_existent_users",
			GroupsTable:   "groups",
			ExpectedError: "no such table: non_existent_users",
		}, {
			Name:          "invalid groups tables",
			UsersTable:    "users",
			GroupsTable:   "non_existent_groups",
			ExpectedError: "no such table: non_existent_groups",
		}, {
			Name:          "invalid users columns",
			UsersTable:    "invalid_users",
			GroupsTable:   "groups",
			ExpectedError: "no such column: password",
		}, {
			Name:          "invalid groups columns",
			UsersTable:    "users",
			GroupsTable:   "invalid_groups",
			ExpectedError: "no such column: group",
		}, {
			Name:        "valid tables",
			UsersTable:  "users",
			GroupsTable: "groups",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewUserDatabase(db, tc.UsersTable, tc.GroupsTable)
			if tc.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.ExpectedError)
			}
		})
	}
}

func TestGetByUsername(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("CREATE TABLE `users` (username TEXT PRIMARY KEY, password TEXT)")
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE `groups` (username TEXT, `group` TEXT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO `users` (username, password) VALUES (\"user1\", \"pass1\")")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO `users` (username, password) VALUES (\"user2\", \"pass2\")")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO `users` (username, password) VALUES (\"user3\", \"pass3\")")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO `groups` (username, `group`) VALUES (\"user2\", \"group1\")")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO `groups` (username, `group`) VALUES (\"user3\", \"group1\")")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO `groups` (username, `group`) VALUES (\"user3\", \"group2\")")
	require.NoError(t, err)
	d, err := NewUserDatabase(db, "users", "groups")
	require.NoError(t, err)

	testCases := []struct {
		Name         string
		Username     string
		ExpectedUser *User
	}{
		{
			Name:         "non existent user",
			Username:     "user99",
			ExpectedUser: nil,
		}, {
			Name:     "user without groups",
			Username: "user1",
			ExpectedUser: &User{
				Username: "user1",
				Password: "pass1",
			},
		}, {
			Name:     "user with one group",
			Username: "user2",
			ExpectedUser: &User{
				Username: "user2",
				Password: "pass2",
				Groups:   []string{"group1"},
			},
		}, {
			Name:     "user with multiple groups",
			Username: "user3",
			ExpectedUser: &User{
				Username: "user3",
				Password: "pass3",
				Groups:   []string{"group1", "group2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			u, err := d.GetByUsername(tc.Username)
			require.NoError(t, err)

			assert.Equal(t, tc.ExpectedUser, u)
		})
	}

}
