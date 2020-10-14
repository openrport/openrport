package chserver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/server/api/users"
)

func TestValidateCredentials(t *testing.T) {
	u1Hashed := &users.User{
		Username: "admin",
		Password: "$2y$05$cIOk1IlsdgdUeZpV464d6OXKI1tF2Yc3MWo55xDu4XhopEJmGb2KC",
		Groups:   []string{"admins", "users", "gods"},
	}
	u2Hashed := &users.User{
		Username: "user2",
		Password: "$2y$10$ZHVHjjP1BnXLtUknPXg4KuRcprfgMjgpe/ZUv5XKwAD74KKuwDjtu",
		Groups:   []string{},
	}
	u1Plaintext := &users.User{
		Username: "admin",
		Password: "foobaz",
	}
	u2Plaintext := &users.User{
		Username: "user2",
		Password: "user2",
	}

	testCases := []struct {
		descr string // Test Case Description

		repoUsers []*users.User
		username  string
		password  string

		wantRes bool
	}{
		{
			descr:     "empty credentials",
			repoUsers: []*users.User{u1Plaintext},
			username:  "",
			password:  "",
			wantRes:   false,
		},
		{
			descr:     "empty username",
			repoUsers: []*users.User{u1Plaintext},
			username:  "",
			password:  u1Plaintext.Password,
			wantRes:   false,
		},
		{
			descr:     "empty password",
			repoUsers: []*users.User{u1Plaintext},
			username:  u1Plaintext.Username,
			password:  "",
			wantRes:   false,
		},
		{
			descr:     "authorized, both Users Repo and credentials with plaintext password",
			repoUsers: []*users.User{u1Plaintext},
			username:  u1Plaintext.Username,
			password:  u1Plaintext.Password,
			wantRes:   true,
		},
		{
			descr:     "unauthorized, both Users Repo and credentials with plaintext password, unknown username",
			repoUsers: []*users.User{u1Plaintext},
			username:  "unknown-username",
			password:  u1Plaintext.Password,
			wantRes:   false,
		},
		{
			descr:     "unauthorized, both Users Repo and credentials with plaintext password, wrong password",
			repoUsers: []*users.User{u1Plaintext},
			username:  u1Plaintext.Username,
			password:  "wrong-password",
			wantRes:   false,
		},
		{
			descr:     "unauthorized, both Users Repo and credentials with plaintext password, mixed credentials",
			repoUsers: []*users.User{u1Plaintext, u2Plaintext},
			username:  u1Plaintext.Username,
			password:  u2Plaintext.Password,
			wantRes:   false,
		},
		{
			descr:     "authorized, Users Repo contains bcrypt hashed passwords and credentials with plaintext password",
			repoUsers: []*users.User{u1Hashed, u2Hashed},
			username:  u1Plaintext.Username,
			password:  u1Plaintext.Password,
			wantRes:   true,
		},
		{
			descr:     "authorized, Users Repo contains bcrypt hashed passwords and credentials with plaintext password, user 2",
			repoUsers: []*users.User{u1Hashed, u2Hashed},
			username:  u2Plaintext.Username,
			password:  u2Plaintext.Password,
			wantRes:   true,
		},
		{
			descr:     "unauthorized, Users Repo contains bcrypt hashed passwords and credentials with plaintext password, mixed credentials",
			repoUsers: []*users.User{u1Hashed, u2Hashed},
			username:  u1Plaintext.Username,
			password:  u2Plaintext.Password,
			wantRes:   false,
		},
		{
			descr:     "unauthorized, user not found",
			repoUsers: []*users.User{u1Plaintext},
			username:  u2Plaintext.Username,
			password:  u2Plaintext.Password,
			wantRes:   false,
		},
	}

	for _, tc := range testCases {
		msg := fmt.Sprintf("test case: %q", tc.descr)

		// given
		al := &APIListener{}
		al.userSrv = users.NewUserRepository(tc.repoUsers)

		// when
		gotRes, gotErr := al.validateCredentials(tc.username, tc.password)

		// then
		assert.NoErrorf(t, gotErr, msg)
		assert.Equalf(t, gotRes, tc.wantRes, msg)
	}
}

func TestIsAuthorizationOn(t *testing.T) {
	for _, tc := range []struct {
		name    string
		config  *Config
		wantRes bool
	}{
		{
			name:    "no config",
			config:  nil,
			wantRes: false,
		},
		{
			name:    "no api auth",
			config:  &Config{},
			wantRes: false,
		},
		{
			name: "api auth file",
			config: &Config{
				API: APIConfig{
					AuthFile: "auth-file",
				},
			},
			wantRes: true,
		},
		{
			name: "api single auth",
			config: &Config{
				API: APIConfig{
					Auth: "user:pswd",
				},
			},
			wantRes: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			al := APIListener{
				Server: &Server{
					config: tc.config,
				},
			}
			assert.Equal(t, tc.wantRes, al.IsAuthorizationOn())
		})
	}
}
