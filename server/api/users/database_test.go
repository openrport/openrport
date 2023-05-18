package users

import (
	"fmt"
	"os"
	"testing"

	plusprm "github.com/realvnc-labs/rport/plus/capabilities/permission"
	chshare "github.com/realvnc-labs/rport/share/logger"

	"github.com/realvnc-labs/rport/share/test"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLog = chshare.NewLogger("client", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)

func TestNewUserDatabase(t *testing.T) {
	testCases := []struct {
		Name              string
		UsersTable        string
		GroupsTable       string
		GroupDetailsTable string
		ExpectedError     string
		twoFAOn           bool
		totPOn            bool
		plusOn            bool
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
			Name:              "invalid groups tables",
			UsersTable:        "users",
			GroupsTable:       "groups",
			GroupDetailsTable: "non_existent_group_details",
			ExpectedError:     "no such table: non_existent_group_details",
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
			Name:              "invalid groups details columns",
			UsersTable:        "users",
			GroupsTable:       "groups",
			GroupDetailsTable: "invalid_group_details",
			ExpectedError:     "no such column: permissions",
		}, {
			Name:              "valid tables",
			UsersTable:        "users",
			GroupsTable:       "groups",
			GroupDetailsTable: "group_details",
		}, {
			Name:        "valid tables - no details",
			UsersTable:  "users",
			GroupsTable: "groups",
		},
		{
			Name:        "totP on",
			UsersTable:  "users",
			GroupsTable: "groups",
			totPOn:      true,
			plusOn:      false,
		},
		{
			Name:        "2fa and totP on",
			UsersTable:  "users",
			GroupsTable: "groups",
			twoFAOn:     true,
			plusOn:      false,
		},
		{
			Name:        "2fa on",
			UsersTable:  "users",
			GroupsTable: "groups",
			twoFAOn:     true,
			totPOn:      true,
			plusOn:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			db, err := sqlx.Connect("sqlite3", ":memory:")
			require.NoError(t, err)
			defer db.Close()

			err = prepareTables(db, tc.twoFAOn, tc.totPOn, tc.plusOn)
			require.NoError(t, err)

			_, err = db.Exec("CREATE TABLE `invalid_users` (username TEXT PRIMARY KEY, pass TEXT)")
			require.NoError(t, err)
			_, err = db.Exec("CREATE TABLE `invalid_groups` (username TEXT, other TEXT)")
			require.NoError(t, err)
			_, err = db.Exec("CREATE TABLE `invalid_group_details` (name TEXT, other TEXT)")
			require.NoError(t, err)

			_, err = NewUserDatabase(db, tc.UsersTable, tc.GroupsTable, tc.GroupDetailsTable, tc.twoFAOn, tc.totPOn, false, testLog)
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

	err = prepareTables(db, true, false, false)
	require.NoError(t, err)

	err = prepareDummyData(db, true, false, false)
	require.NoError(t, err)

	d, err := NewUserDatabase(db, "users", "groups", "", false, false, false, testLog)
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
				Username:        "user1",
				Password:        "pass1",
				PasswordExpired: PasswordExpired(false),
			},
		}, {
			Name:     "user with one group",
			Username: "user2",
			ExpectedUser: &User{
				Username:        "user2",
				Password:        "pass2",
				PasswordExpired: PasswordExpired(false),
				Groups:          []string{"group1"},
			},
		}, {
			Name:     "user with multiple groups",
			Username: "user3",
			ExpectedUser: &User{
				Username:        "user3",
				Password:        "pass3",
				PasswordExpired: PasswordExpired(false),
				Groups:          []string{"group1", "group2"},
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

	err = prepareTables(db, false, true, false)
	require.NoError(t, err)

	err = prepareDummyData(db, false, true, false)
	require.NoError(t, err)

	d, err := NewUserDatabase(db, "users", "groups", "", false, true, false, testLog)
	require.NoError(t, err)

	actualUsers, err := d.GetAll()
	require.NoError(t, err)

	expectedUsers := []*User{
		{
			Username:        "user1",
			Password:        "pass1",
			PasswordExpired: PasswordExpired(false),
			Groups:          nil,
			TotP:            "totP123",
		},
		{
			Username:        "user2",
			Password:        "pass2",
			PasswordExpired: PasswordExpired(false),
			Groups: []string{
				"group1",
			},
		},
		{
			Username:        "user3",
			Password:        "pass3",
			PasswordExpired: PasswordExpired(false),
			Groups: []string{
				"group1",
				"group2",
			},
		},
	}
	assert.Equal(t, expectedUsers, actualUsers)
}

func TestListGroups(t *testing.T) {
	testCases := []struct {
		Name         string
		DetailsTable string
		Expected     []Group
	}{
		{
			Name:         "no details",
			DetailsTable: "",
			Expected: []Group{
				NewGroup("group1", nil, nil),
				NewGroup("group2", nil, nil),
			},
		},
		{
			Name:         "with details",
			DetailsTable: "group_details",
			Expected: []Group{
				NewGroup("group1", nil, nil, PermissionCommands),
				NewGroup("group2", nil, nil),
				NewGroup("group3", nil, nil, PermissionScripts),
			},
		},
	}

	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db, false, false, false)
	require.NoError(t, err)

	err = prepareDummyData(db, false, false, false)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			d, err := NewUserDatabase(db, "users", "groups", tc.DetailsTable, false, false, false, testLog)
			require.NoError(t, err)

			actualGroups, err := d.ListGroups()
			require.NoError(t, err)

			assert.ElementsMatch(t, tc.Expected, actualGroups)
		})
	}
}

func TestListGroupsPlusOn(t *testing.T) {
	testCases := []struct {
		Name         string
		DetailsTable string
		Expected     []Group
	}{
		{
			Name:         "with details",
			DetailsTable: "group_details",
			Expected: []Group{
				NewGroup("group1", nil, nil, PermissionCommands),
				NewGroup("group2", nil, nil),
				NewGroup("group3", nil, nil, PermissionScripts),
				NewGroup("group4",
					&plusprm.StringInterfaceMap{
						"auth_allowed":         true,
						"local":                []interface{}{"20000", "20001"},
						"auto_close":           map[string]interface{}{"max": "60m", "min": "2m"},
						"idle-timeout-minutes": map[string]interface{}{"min": 5.0},
					},
					nil,
					PermissionTunnels),
				NewGroup("group5",
					nil,
					&plusprm.StringInterfaceMap{
						"deny":    []interface{}{"apache2", "ssh"},
						"is_sudo": false,
					},
					PermissionCommands),
			},
		},
	}

	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db, false, false, true)
	require.NoError(t, err)

	err = prepareDummyData(db, false, false, true)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			d, err := NewUserDatabase(db, "users", "groups", tc.DetailsTable, false, false, true, testLog)
			require.NoError(t, err)

			actualGroups, err := d.ListGroups()
			require.NoError(t, err)

			assert.ElementsMatch(t, tc.Expected, actualGroups)
		})
	}
}

func TestGetGroup(t *testing.T) {
	testCases := []struct {
		Name         string
		DetailsTable string
		Group        string
		Expected     Group
	}{
		{
			Name:         "no details",
			DetailsTable: "",
			Group:        "group1",
			Expected:     NewGroup("group1", nil, nil),
		},
		{
			Name:         "with details existing",
			DetailsTable: "group_details",
			Group:        "group1",
			Expected:     NewGroup("group1", nil, nil, PermissionCommands),
		},
		{
			Name:         "with details not existing",
			DetailsTable: "group_details",
			Group:        "group2",
			Expected:     NewGroup("group2", nil, nil),
		},
	}

	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db, false, false, false)
	require.NoError(t, err)

	err = prepareDummyData(db, false, false, false)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			d, err := NewUserDatabase(db, "users", "groups", tc.DetailsTable, false, false, false, testLog)
			require.NoError(t, err)

			actual, err := d.GetGroup(tc.Group)
			require.NoError(t, err)

			assert.Equal(t, tc.Expected, actual)
		})
	}
}

func TestGetGroupPlusOn(t *testing.T) {
	testCases := []struct {
		Name         string
		DetailsTable string
		Group        string
		Expected     Group
	}{
		{
			Name:         "no details",
			DetailsTable: "",
			Group:        "group1",
			Expected:     NewGroup("group1", nil, nil),
		},
		{
			Name:         "with details existing",
			DetailsTable: "group_details",
			Group:        "group1",
			Expected:     NewGroup("group1", nil, nil, PermissionCommands),
		},
		{
			Name:         "with details not existing",
			DetailsTable: "group_details",
			Group:        "group4",
			Expected: NewGroup("group4",
				&plusprm.StringInterfaceMap{
					"auth_allowed":         true,
					"local":                []interface{}{"20000", "20001"},
					"auto_close":           map[string]interface{}{"max": "60m", "min": "2m"},
					"idle-timeout-minutes": map[string]interface{}{"min": 5.0},
				},
				nil,
				PermissionTunnels),
		},
		{
			Name:         "with details not existing",
			DetailsTable: "group_details",
			Group:        "group5",
			Expected: NewGroup("group5",
				nil,
				&plusprm.StringInterfaceMap{
					"deny":    []interface{}{"apache2", "ssh"},
					"is_sudo": false,
				},
				PermissionCommands),
		},
	}

	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db, false, false, true)
	require.NoError(t, err)

	err = prepareDummyData(db, false, false, true)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			d, err := NewUserDatabase(db, "users", "groups", tc.DetailsTable, false, false, true, testLog)
			require.NoError(t, err)

			actual, err := d.GetGroup(tc.Group)
			require.NoError(t, err)

			assert.Equal(t, tc.Expected, actual)
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	testCases := []struct {
		Name string
		Group
	}{
		{
			Name:  "existing",
			Group: NewGroup("group1", nil, nil, PermissionScripts),
		},
		{
			Name:  "with details not existing",
			Group: NewGroup("group2", nil, nil),
		},
	}

	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db, false, false, false)
	require.NoError(t, err)

	err = prepareDummyData(db, false, false, false)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			d, err := NewUserDatabase(db, "users", "groups", "group_details", false, false, false, testLog)
			require.NoError(t, err)

			err = d.UpdateGroup(tc.Group.Name, tc.Group)
			require.NoError(t, err)

			actual, err := d.GetGroup(tc.Group.Name)
			require.NoError(t, err)

			assert.Equal(t, tc.Group, actual)
		})
	}
}

func TestUpdateGroupPlusOn(t *testing.T) {
	testCases := []struct {
		Name string
		Group
		ExpectedError string
	}{
		{
			Name: "existing",
			Group: NewGroup("group1",
				&plusprm.StringInterfaceMap{
					"auth_allowed":         true,
					"local":                []interface{}{"20001", "20002"},
					"auto_close":           map[string]interface{}{"max": "30m", "min": "2m"},
					"idle-timeout-minutes": map[string]interface{}{"min": 2.0},
				},
				&plusprm.StringInterfaceMap{
					"deny":    []interface{}{"apache2", "ssh"},
					"is_sudo": false,
				},
				PermissionCommands, PermissionTunnels),
		},
		{
			Name: "existing only tunnels",
			Group: NewGroup("group4",
				&plusprm.StringInterfaceMap{
					"auth_allowed":         true,
					"local":                []interface{}{"20001", "20002"},
					"auto_close":           map[string]interface{}{"max": "30m", "min": "2m"},
					"idle-timeout-minutes": map[string]interface{}{"min": 2.0},
				}, nil, PermissionTunnels),
		},
		{
			Name: "wrong format strings and ints",
			Group: NewGroup("group4",
				&plusprm.StringInterfaceMap{
					"local": []interface{}{20001, "20002"},
				}, nil),
			ExpectedError: `sql: converting argument $3 type: invalid restriction list 20001 of type int`,
		},
		{
			Name: "wrong rule in max / min values",
			Group: NewGroup("group4",
				&plusprm.StringInterfaceMap{
					"idle-timeout-Minutes": map[string]interface{}{"mex": 2.0},
				}, nil),
			ExpectedError: `sql: converting argument $3 type: invalid restriction rule 'mex'`,
		},
		{
			Name: "unparseable duration",
			Group: NewGroup("group4",
				&plusprm.StringInterfaceMap{
					"idle-timeout-minutes": map[string]interface{}{"max": "jk"},
				}, nil),
			ExpectedError: `sql: converting argument $3 type: restriction jk not parseable as time.duration: invalid type`,
		},
		{
			Name: "unparseable regex single string",
			Group: NewGroup("group4",
				&plusprm.StringInterfaceMap{
					"host_header": "[abc",
				}, nil),
			ExpectedError: "sql: converting argument $3 type: invalid restriction regular expression \"[abc\": error parsing regexp: missing closing ]: `[abc`",
		},
		{
			Name: "unparseable regex in a list of strings allow / deny",
			Group: NewGroup("group4",
				&plusprm.StringInterfaceMap{
					"Deny": []interface{}{"abc", "[abc"},
				}, nil),
			ExpectedError: "sql: converting argument $3 type: invalid restriction regular expression \"[abc\": error parsing regexp: missing closing ]: `[abc`",
		},
		{
			Name: "wrong type as restriction",
			Group: NewGroup("group4",
				&plusprm.StringInterfaceMap{
					"hello": 7.0,
				}, nil),
			ExpectedError: "sql: converting argument $3 type: restriction 7 of type float64 not recognized",
		},
	}

	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db, false, false, true)
	require.NoError(t, err)

	err = prepareDummyData(db, false, false, true)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			d, err := NewUserDatabase(db, "users", "groups", "group_details", false, false, true, testLog)
			require.NoError(t, err)

			err = d.UpdateGroup(tc.Group.Name, tc.Group)
			if tc.ExpectedError != "" {
				require.EqualError(t, err, tc.ExpectedError)
			} else {
				require.NoError(t, err)
				actual, err := d.GetGroup(tc.Group.Name)
				require.NoError(t, err)

				assert.Equal(t, tc.Group, actual)
			}
		})
	}
}

func TestDeleteGroup(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = prepareTables(db, false, false, false)
	require.NoError(t, err)

	err = prepareDummyData(db, false, false, false)
	require.NoError(t, err)

	d, err := NewUserDatabase(db, "users", "groups", "group_details", false, false, false, testLog)
	require.NoError(t, err)

	err = d.DeleteGroup("group1")
	require.NoError(t, err)

	actual, err := d.ListGroups()
	require.NoError(t, err)

	expected := []Group{
		NewGroup("group2", nil, nil),
		NewGroup("group3", nil, nil, PermissionScripts),
	}
	assert.ElementsMatch(t, expected, actual)
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
					"username":         "login1",
					"password":         "pass1",
					"password_expired": false,
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

			err = prepareTables(db, false, false, false)
			require.NoError(t, err)

			d, err := NewUserDatabase(db, "users", "groups", "", false, false, false, testLog)
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
					"username":         "user2",
					"password":         "pass2",
					"password_expired": false,
				},
				{
					"username":         "user3",
					"password":         "pass3",
					"password_expired": false,
				},
				{
					"username":         "user_one",
					"password":         "pass_one",
					"password_expired": false,
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
					"username":         "user1",
					"password":         "pass1",
					"password_expired": false,
				},
				{
					"username":         "user2",
					"password":         "pass_two",
					"password_expired": false,
				},
				{
					"username":         "user3",
					"password":         "pass3",
					"password_expired": false,
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
					"username":         "user1",
					"password":         "pass1",
					"password_expired": false,
				},
				{
					"username":         "user2",
					"password":         "pass2",
					"password_expired": false,
				},
				{
					"username":         "user3",
					"password":         "pass3",
					"password_expired": false,
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
					"username":         "user1",
					"password":         "pass1",
					"password_expired": false,
				},
				{
					"username":         "user2",
					"password":         "pass2",
					"password_expired": false,
				},
				{
					"username":         "user3",
					"password":         "pass3",
					"password_expired": false,
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

			err = prepareTables(db, false, false, false)
			require.NoError(t, err)

			err = prepareDummyData(db, false, false, false)
			require.NoError(t, err)

			d, err := NewUserDatabase(db, "users", "groups", "", false, false, false, testLog)
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

	err = prepareTables(db, false, false, false)
	require.NoError(t, err)

	err = prepareDummyData(db, false, false, false)
	require.NoError(t, err)

	d, err := NewUserDatabase(db, "users", "groups", "", false, false, false, testLog)
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

func prepareTables(db *sqlx.DB, twoFAOn, totPON bool, plusOn bool) error {
	q := "CREATE TABLE `users` (username TEXT PRIMARY KEY, password TEXT, password_expired BOOLEAN NOT NULL CHECK (password_expired IN (0, 1)) DEFAULT 0%s)"
	dynamicFieldsQ := ""
	if twoFAOn {
		dynamicFieldsQ += ", two_fa_send_to TEXT NOT NULL DEFAULT ''"
	}
	if totPON {
		dynamicFieldsQ += ", totp_secret TEXT NOT NULL DEFAULT ''"
	}
	q = fmt.Sprintf(q, dynamicFieldsQ)

	_, err := db.Exec(q)
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE `groups` (username TEXT, `group` TEXT)")
	if err != nil {
		return err
	}

	if plusOn {
		_, err = db.Exec(`CREATE TABLE "group_details" (name TEXT, permissions TEXT, tunnels_restricted TEXT, commands_restricted TEXT);CREATE UNIQUE INDEX "main"."group_details_name" ON "group_details" ("name" ASC);`)
	} else {
		_, err = db.Exec(`CREATE TABLE "group_details" (name TEXT, permissions TEXT);CREATE UNIQUE INDEX "main"."group_details_name" ON "group_details" ("name" ASC);`)
	}

	if err != nil {
		return err
	}

	return nil
}

func prepareDummyData(db *sqlx.DB, withTwoFA, withTotP bool, plusOn bool) error {
	var err error
	if !withTotP {
		_, err = db.Exec("INSERT INTO `users` (username, password) VALUES (\"user1\", \"pass1\")")
		if err != nil {
			return err
		}
	} else {
		_, err = db.Exec("INSERT INTO `users` (username, password, totp_secret) VALUES (\"user1\", \"pass1\", \"totP123\")")
		if err != nil {
			return err
		}
	}

	if !withTwoFA {
		_, err = db.Exec("INSERT INTO `users` (username, password) VALUES (\"user2\", \"pass2\")")
		if err != nil {
			return err
		}
	} else {
		_, err = db.Exec("INSERT INTO `users` (username, password, two_fa_send_to) VALUES (\"user2\", \"pass2\", \"no@mail.me\")")
		if err != nil {
			return err
		}
	}

	if plusOn {
		_, err = db.Exec(`INSERT INTO group_details (name, permissions, tunnels_restricted) VALUES ("group4", '{"tunnels":true}', '{ "local": ["20000","20001"], "idle-timeout-minutes": { "min": 5 }, "auto_close": { "max": "60m", "min" : "2m" }, "auth_allowed": true }')`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`INSERT INTO group_details (name, permissions, commands_restricted) VALUES ("group5", '{"commands":true}', '{ "deny": ["apache2","ssh"], "is_sudo": false }' )`)
		if err != nil {
			return err
		}

		_, err = db.Exec("INSERT INTO `groups` (username, `group`) VALUES (\"user4\", \"group4\")")
		if err != nil {
			return err
		}

		_, err = db.Exec("INSERT INTO `groups` (username, `group`) VALUES (\"user5\", \"group5\")")
		if err != nil {
			return err
		}

	}

	_, err = db.Exec(`INSERT INTO group_details (name, permissions) VALUES ("group1", '{"commands":true}')`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO group_details (name, permissions) VALUES ("group3", '{"scripts":true}')`)
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
	query := fmt.Sprintf("SELECT `username`, `password`, `password_expired` FROM `%s` order by `username`", usersTableName)
	test.AssertRowsEqual(t, db, expectedRows, query, []interface{}{})
}

func assertGroupTableEquals(t *testing.T, db *sqlx.DB, groupTableName string, expectedRows []map[string]interface{}) {
	query := fmt.Sprintf("SELECT `username`, `group` FROM `%s` order by `username`, `group`", groupTableName)
	test.AssertRowsEqual(t, db, expectedRows, query, []interface{}{})
}
