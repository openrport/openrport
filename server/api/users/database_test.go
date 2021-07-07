package users

import (
	"fmt"
	"os"
	"testing"

	"github.com/cloudradar-monitoring/rport/share/test"

	chshare "github.com/cloudradar-monitoring/rport/share"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLog = chshare.NewLogger("client", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)

func TestNewUserDatabase(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db)
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
			_, err := NewUserDatabase(db, tc.UsersTable, tc.GroupsTable, false, testLog)
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

	err = prepareTables(db)
	require.NoError(t, err)

	err = prepareDummyData(db)
	require.NoError(t, err)

	d, err := NewUserDatabase(db, "users", "groups", false, testLog)
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

func TestGetAll(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db)
	require.NoError(t, err)

	err = prepareDummyData(db)
	require.NoError(t, err)

	d, err := NewUserDatabase(db, "users", "groups", false, testLog)
	require.NoError(t, err)

	actualUsers, err := d.GetAll()
	require.NoError(t, err)

	expectedUsers := []*User{
		{
			Username: "user1",
			Password: "pass1",
			Groups:   nil,
		},
		{
			Username: "user2",
			Password: "pass2",
			Groups: []string{
				"group1",
			},
		},
		{
			Username: "user3",
			Password: "pass3",
			Groups: []string{
				"group1",
				"group2",
			},
		},
	}
	assert.Equal(t, expectedUsers, actualUsers)
}

func TestGetAllGroups(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db)
	require.NoError(t, err)

	err = prepareDummyData(db)
	require.NoError(t, err)

	d, err := NewUserDatabase(db, "users", "groups", false, testLog)
	require.NoError(t, err)

	actualGroups, err := d.GetAllGroups()
	require.NoError(t, err)

	expectedGroups := []string{
		"group1",
		"group2",
	}
	assert.Equal(t, expectedGroups, actualGroups)
}

func TestAdd(t *testing.T) {
	testCases := []struct {
		name              string
		userToChange      *User
		expectedUserRows  []map[string]interface{}
		expectedGroupRows []map[string]interface{}
	}{
		{
			name: "create user",
			userToChange: &User{
				Username: "login1",
				Password: "pass1",
				Groups: []string{
					"group1",
					"group2",
				},
			},
			expectedUserRows: []map[string]interface{}{
				{
					"username": "login1",
					"password": "pass1",
				},
			},
			expectedGroupRows: []map[string]interface{}{
				{
					"username": "login1",
					"group":    "group1",
				},
				{
					"username": "login1",
					"group":    "group2",
				},
			},
		},
	}

	for i := range testCases {
		t.Run(testCases[i].name, func(t *testing.T) {
			testCase := testCases[i]

			db, err := sqlx.Connect("sqlite3", ":memory:")
			require.NoError(t, err)
			defer db.Close()

			err = prepareTables(db)
			require.NoError(t, err)

			d, err := NewUserDatabase(db, "users", "groups", false, testLog)
			require.NoError(t, err)

			err = d.Add(testCase.userToChange)
			require.NoError(t, err)

			assertUserTableEquals(t, d.db, d.usersTableName, testCase.expectedUserRows)
			assertGroupTableEquals(t, d.db, d.groupsTableName, testCase.expectedGroupRows)
		})
	}
}

func TestUpdate(t *testing.T) {
	testCases := []struct {
		name              string
		userToChange      *User
		username          string
		expectedUserRows  []map[string]interface{}
		expectedGroupRows []map[string]interface{}
	}{
		{
			name: "overwrite all fields",
			userToChange: &User{
				Username: "user_one",
				Password: "pass_one",
				Groups: []string{
					"group1",
				},
			},
			username: "user1",
			expectedUserRows: []map[string]interface{}{
				{
					"username": "user2",
					"password": "pass2",
				},
				{
					"username": "user3",
					"password": "pass3",
				},
				{
					"username": "user_one",
					"password": "pass_one",
				},
			},
			expectedGroupRows: []map[string]interface{}{
				{
					"username": "user2",
					"group":    "group1",
				},
				{
					"username": "user3",
					"group":    "group1",
				},
				{
					"username": "user3",
					"group":    "group2",
				},
				{
					"username": "user_one",
					"group":    "group1",
				},
			},
		},
		{
			name: "overwrite pass",
			userToChange: &User{
				Password: "pass_two",
			},
			username: "user2",
			expectedUserRows: []map[string]interface{}{
				{
					"username": "user1",
					"password": "pass1",
				},
				{
					"username": "user2",
					"password": "pass_two",
				},
				{
					"username": "user3",
					"password": "pass3",
				},
			},
			expectedGroupRows: []map[string]interface{}{
				{
					"username": "user2",
					"group":    "group1",
				},
				{
					"username": "user3",
					"group":    "group1",
				},
				{
					"username": "user3",
					"group":    "group2",
				},
			},
		},
		{
			name: "overwrite groups",
			userToChange: &User{
				Groups: []string{
					"group1",
				},
			},
			username: "user3",
			expectedUserRows: []map[string]interface{}{
				{
					"username": "user1",
					"password": "pass1",
				},
				{
					"username": "user2",
					"password": "pass2",
				},
				{
					"username": "user3",
					"password": "pass3",
				},
			},
			expectedGroupRows: []map[string]interface{}{
				{
					"username": "user2",
					"group":    "group1",
				},
				{
					"username": "user3",
					"group":    "group1",
				},
			},
		},
		{
			name: "empty groups",
			userToChange: &User{
				Groups: []string{},
			},
			username: "user3",
			expectedUserRows: []map[string]interface{}{
				{
					"username": "user1",
					"password": "pass1",
				},
				{
					"username": "user2",
					"password": "pass2",
				},
				{
					"username": "user3",
					"password": "pass3",
				},
			},
			expectedGroupRows: []map[string]interface{}{
				{
					"username": "user2",
					"group":    "group1",
				},
			},
		},
	}

	for i := range testCases {
		t.Run(testCases[i].name, func(t *testing.T) {
			db, err := sqlx.Connect("sqlite3", ":memory:")
			require.NoError(t, err)
			defer db.Close()

			err = prepareTables(db)
			require.NoError(t, err)

			err = prepareDummyData(db)
			require.NoError(t, err)

			d, err := NewUserDatabase(db, "users", "groups", false, testLog)
			require.NoError(t, err)

			testCase := testCases[i]

			err = d.Update(testCase.userToChange, testCase.username)
			require.NoError(t, err)

			assertUserTableEquals(t, d.db, d.usersTableName, testCase.expectedUserRows)
			assertGroupTableEquals(t, d.db, d.groupsTableName, testCase.expectedGroupRows)
		})
	}
}

func TestDelete(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db)
	require.NoError(t, err)

	err = prepareDummyData(db)
	require.NoError(t, err)

	d, err := NewUserDatabase(db, "users", "groups", false, testLog)
	require.NoError(t, err)

	err = d.Delete("user1")
	require.NoError(t, err)

	err = d.Delete("user2")
	require.NoError(t, err)

	err = d.Delete("user3")
	require.NoError(t, err)

	assertUserTableEquals(t, db, d.usersTableName, []map[string]interface{}{})
	assertGroupTableEquals(t, db, d.groupsTableName, []map[string]interface{}{})
}

func prepareTables(db *sqlx.DB) error {
	_, err := db.Exec("CREATE TABLE `users` (username TEXT PRIMARY KEY, password TEXT)")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE `groups` (username TEXT, `group` TEXT)")
	if err != nil {
		return err
	}

	return nil
}

func prepareDummyData(db *sqlx.DB) error {
	_, err := db.Exec("INSERT INTO `users` (username, password) VALUES (\"user1\", \"pass1\")")
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO `users` (username, password) VALUES (\"user2\", \"pass2\")")
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO `users` (username, password) VALUES (\"user3\", \"pass3\")")
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO `groups` (username, `group`) VALUES (\"user2\", \"group1\")")
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO `groups` (username, `group`) VALUES (\"user3\", \"group1\")")
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO `groups` (username, `group`) VALUES (\"user3\", \"group2\")")
	if err != nil {
		return err
	}

	return nil
}

func assertUserTableEquals(t *testing.T, db *sqlx.DB, usersTableName string, expectedRows []map[string]interface{}) {
	query := fmt.Sprintf("SELECT `username`, `password` FROM `%s` order by `username`", usersTableName)
	test.AssertRowsEqual(t, db, expectedRows, query, []interface{}{})
}

func assertGroupTableEquals(t *testing.T, db *sqlx.DB, groupTableName string, expectedRows []map[string]interface{}) {
	query := fmt.Sprintf("SELECT `username`, `group` FROM `%s` order by `username`, `group`", groupTableName)
	test.AssertRowsEqual(t, db, expectedRows, query, []interface{}{})
}
