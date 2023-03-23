package users

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/share/logger"
)

func TestParseUsers(t *testing.T) {
	u1JSON := `{
    "username": "user1",
    "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
    "groups": [
      "admins",
      "users",
      "gods"
    ]
  }`
	u1 := &User{
		Username: "user1",
		Password: "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
		Groups:   []string{"admins", "users", "gods"},
	}

	u2JSON := `{
    "username": "user2",
    "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
    "groups": []
  }`
	u2 := &User{
		Username: "user2",
		Password: "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
		Groups:   []string{},
	}

	u3JSON := `{
    "username": "user3",
    "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu"
  }`
	u3 := &User{
		Username: "user3",
		Password: "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
		Groups:   nil,
	}

	testCases := []struct {
		descr string // Test Case Description

		jsonBytes string

		wantRes         []*User
		wantErrContains string // part of an expected error
	}{
		{
			descr:           "empty file",
			jsonBytes:       ``,
			wantRes:         nil,
			wantErrContains: "",
		},
		{
			descr:           "empty JSON array",
			jsonBytes:       `[]`,
			wantRes:         nil,
			wantErrContains: "",
		},
		{
			descr:           "3 users",
			jsonBytes:       fmt.Sprintf("[%s,%s,%s]", u1JSON, u2JSON, u3JSON),
			wantRes:         []*User{u1, u2, u3},
			wantErrContains: "",
		},
		{
			descr:           "user with not trimmed whitespaces",
			jsonBytes:       `[{"username": "  user3   ","password": " $2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu "}]`,
			wantRes:         []*User{u3},
			wantErrContains: "",
		},
		{
			descr:           "username is missing",
			jsonBytes:       `[{"password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu", "groups":[]}]`,
			wantRes:         nil,
			wantErrContains: "username can not be empty",
		},
		{
			descr:           "empty username",
			jsonBytes:       `[{"username": "","password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu"}]`,
			wantRes:         nil,
			wantErrContains: "username can not be empty",
		},
		{
			descr:           "username with whitespaces",
			jsonBytes:       `[{"username": "     ","password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu"}]`,
			wantRes:         nil,
			wantErrContains: "username can not be empty",
		},
		{
			descr:           "password is missing",
			jsonBytes:       `[{"username": "admin", "groups":[]}]`,
			wantRes:         nil,
			wantErrContains: "password can not be empty",
		},
		{
			descr:           "empty password",
			jsonBytes:       `[{"username": "admin","password": ""}]`,
			wantRes:         nil,
			wantErrContains: "password can not be empty",
		},
		{
			descr:           "password with whitespaces",
			jsonBytes:       `[{"username": "admin","password": "  "}]`,
			wantRes:         nil,
			wantErrContains: "password can not be empty",
		},
		{
			descr:           "non unique username",
			jsonBytes:       fmt.Sprintf("[%s,%s]", u1JSON, u1JSON),
			wantRes:         nil,
			wantErrContains: "non unique username",
		},
		{
			descr:           "plaintext password instead of bcrypt hashed",
			jsonBytes:       `[{"username": "admin","password": "admin"}]`,
			wantRes:         nil,
			wantErrContains: "require passwords to be bcrypt hashed and to be compatible with",
		},
		{
			descr:           "corrupted json",
			jsonBytes:       `afsdf saf234 sdfe4r`,
			wantRes:         nil,
			wantErrContains: "failed to parse users data",
		},
		{
			descr:           "partially corrupted json at the end",
			jsonBytes:       `[{"username": "admin","password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu"},["lala"]`,
			wantRes:         nil,
			wantErrContains: "failed to parse user",
		},
		{
			descr:           "valid json + trash at the end",
			jsonBytes:       fmt.Sprintf("[%s,%s], %s", u1JSON, u3JSON, `["some trash"]`),
			wantRes:         []*User{u1, u3},
			wantErrContains: "",
		},
	}

	for _, tc := range testCases {
		msg := fmt.Sprintf("test case: %q", tc.descr)

		// given
		fileMock := strings.NewReader(tc.jsonBytes)

		// when
		gotRes, gotErr := parseUsers(fileMock)

		// then
		if len(tc.wantErrContains) > 0 {
			require.Errorf(t, gotErr, msg)
			assert.Containsf(t, gotErr.Error(), tc.wantErrContains, msg)
		} else {
			assert.NoErrorf(t, gotErr, msg)
		}
		assert.Lenf(t, gotRes, len(tc.wantRes), msg)
		assert.ElementsMatch(t, gotRes, tc.wantRes, msg)
	}
}

func TestSaveUsersToFile(t *testing.T) {
	givenUsers := []*User{
		{
			Username: "user1",
			Password: HtpasswdBcryptPrefix + "pass1",
			Groups:   []string{"group1"},
		},
		{
			Username: "user2",
			Password: HtpasswdBcryptPrefix + "pass2",
			Groups:   []string{"group1", "group2"},
		},
	}

	tmpfile, err := ioutil.TempFile("", "example")
	require.NoError(t, err)

	fm := FileManager{
		Logger:         logger.NewLogger("test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug),
		FileName:       tmpfile.Name(),
		FileAccessLock: sync.Mutex{},
	}

	defer func() {
		err := os.Remove(tmpfile.Name())
		require.NoError(t, err)
	}()

	err = fm.SaveUsersToFile(givenUsers)
	require.NoError(t, err)

	actualUsers, err := fm.ReadUsersFromFile()
	require.NoError(t, err)

	assert.Equal(t, givenUsers, actualUsers)
}
