package chserver

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var defaultValidMinServerConfig = ServerConfig{
	URL:     "http://localhost/",
	DataDir: "./",
	Auth:    "abc:def",
}

func TestDatabaseParseAndValidate(t *testing.T) {
	testCases := []struct {
		Name           string
		Database       DatabaseConfig
		ExpectedDriver string
		ExpectedDSN    string
		ExpectedError  error
	}{
		{
			Name: "no db configured",
			Database: DatabaseConfig{
				Type: "",
			},
		}, {
			Name: "invalid type",
			Database: DatabaseConfig{
				Type: "mongodb",
			},
			ExpectedError: errors.New("invalid 'db_type', expected 'mysql' or 'sqlite', got \"mongodb\""),
		}, {
			Name: "sqlite",
			Database: DatabaseConfig{
				Type: "sqlite",
				Name: "/var/lib/rport/rport.db",
			},
			ExpectedDriver: "sqlite3",
			ExpectedDSN:    "/var/lib/rport/rport.db",
		}, {
			Name: "mysql defaults",
			Database: DatabaseConfig{
				Type: "mysql",
			},
			ExpectedDriver: "mysql",
			ExpectedDSN:    "/",
		}, {
			Name: "mysql socket",
			Database: DatabaseConfig{
				Type: "mysql",
				Host: "socket:/var/lib/mysql.sock",
				Name: "testdb",
			},
			ExpectedDriver: "mysql",
			ExpectedDSN:    "unix(/var/lib/mysql.sock)/testdb",
		}, {
			Name: "mysql host",
			Database: DatabaseConfig{
				Type: "mysql",
				Host: "127.0.0.1:3306",
				Name: "testdb",
			},
			ExpectedDriver: "mysql",
			ExpectedDSN:    "tcp(127.0.0.1:3306)/testdb",
		}, {
			Name: "mysql host with user and password",
			Database: DatabaseConfig{
				Type:     "mysql",
				Host:     "127.0.0.1:3306",
				Name:     "testdb",
				User:     "user",
				Password: "password",
			},
			ExpectedDriver: "mysql",
			ExpectedDSN:    "user:password@tcp(127.0.0.1:3306)/testdb",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.Database.ParseAndValidate()
			assert.Equal(t, tc.ExpectedError, err)
			assert.Equal(t, tc.ExpectedDriver, tc.Database.driver)
			assert.Equal(t, tc.ExpectedDSN, tc.Database.dsn)
		})
	}
}

func TestParseAndValidateClientAuth(t *testing.T) {
	testCases := []struct {
		Name                 string
		Config               Config
		ExpectedAuthID       string
		ExpectedAuthPassword string
		ExpectedError        error
	}{
		{
			Name:          "no auth",
			Config:        Config{},
			ExpectedError: errors.New("client authentication must be enabled: set either 'auth', 'auth_file' or 'auth_table'"),
		}, {
			Name: "auth and auth_file",
			Config: Config{
				Server: ServerConfig{
					Auth:     "abc:def",
					AuthFile: "test.json",
				},
			},
			ExpectedError: errors.New("'auth_file' and 'auth' are both set: expected only one of them"),
		}, {
			Name: "auth and auth_table",
			Config: Config{
				Server: ServerConfig{
					Auth:      "abc:def",
					AuthTable: "clients",
				},
			},
			ExpectedError: errors.New("'auth' and 'auth_table' are both set: expected only one of them"),
		}, {
			Name: "auth_table and auth_file",
			Config: Config{
				Server: ServerConfig{
					AuthTable: "clients",
					AuthFile:  "test.json",
				},
			},
			ExpectedError: errors.New("'auth_file' and 'auth_table' are both set: expected only one of them"),
		}, {
			Name: "auth_table without db",
			Config: Config{
				Server: ServerConfig{
					AuthTable: "clients",
				},
			},
			ExpectedError: errors.New("'db_type' must be set when 'auth_table' is set"),
		}, {
			Name: "invalid auth",
			Config: Config{
				Server: ServerConfig{
					Auth: "abc",
				},
			},
			ExpectedError: errors.New("invalid client auth credentials, expected '<client-id>:<password>', got \"abc\""),
		}, {
			Name: "valid auth",
			Config: Config{
				Server: ServerConfig{
					Auth: "abc:def",
				},
			},
			ExpectedAuthID:       "abc",
			ExpectedAuthPassword: "def",
		}, {
			Name: "valid auth_file",
			Config: Config{
				Server: ServerConfig{
					AuthFile: "test.json",
				},
			},
		}, {
			Name: "valid auth_table",
			Config: Config{
				Server: ServerConfig{
					AuthTable: "clients",
				},
				Database: DatabaseConfig{
					Type: "sqlite",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.Config.parseAndValidateClientAuth()
			assert.Equal(t, tc.ExpectedError, err)
			assert.Equal(t, tc.ExpectedAuthID, tc.Config.Server.authID)
			assert.Equal(t, tc.ExpectedAuthPassword, tc.Config.Server.authPassword)
		})
	}
}

func TestParseAndValidateAPI(t *testing.T) {
	testCases := []struct {
		Name                 string
		Config               Config
		ExpectedAuthID       string
		ExpectedAuthPassword string
		ExpectedJwtSecret    bool
		ExpectedError        error
	}{
		{
			Name:          "api disabled, no auth",
			Config:        Config{},
			ExpectedError: nil,
		}, {
			Name: "api disabled, doc_root specified",
			Config: Config{
				API: APIConfig{
					DocRoot: "/var/lib/rport/",
				},
			},
			ExpectedError: errors.New("API: to use document root you need to specify API address"),
		}, {
			Name: "api enabled, no auth",
			Config: Config{
				API: APIConfig{
					Address: "0.0.0.0:3000",
				},
			},
			ExpectedError: errors.New("API: authentication must be enabled: set either 'auth', 'auth_file' or 'auth_user_table'"),
		}, {
			Name: "api enabled, auth and auth_file",
			Config: Config{
				API: APIConfig{
					Address:  "0.0.0.0:3000",
					Auth:     "abc:def",
					AuthFile: "test.json",
				},
			},
			ExpectedError: errors.New("API: 'auth_file' and 'auth' are both set: expected only one of them"),
		}, {
			Name: "api enabled, auth and auth_user_table",
			Config: Config{
				API: APIConfig{
					Address:        "0.0.0.0:3000",
					Auth:           "abc:def",
					AuthUserTable:  "users",
					AuthGroupTable: "groups",
				},
			},
			ExpectedError: errors.New("API: 'auth_user_table' and 'auth' are both set: expected only one of them"),
		}, {
			Name: "api enabled, auth_user_table and auth_file",
			Config: Config{
				API: APIConfig{
					Address:        "0.0.0.0:3000",
					AuthFile:       "test.json",
					AuthUserTable:  "users",
					AuthGroupTable: "groups",
				},
			},
			ExpectedError: errors.New("API: 'auth_user_table' and 'auth_file' are both set: expected only one of them"),
		}, {
			Name: "api enabled, auth_user_table without auth_group_table",
			Config: Config{
				API: APIConfig{
					Address:       "0.0.0.0:3000",
					AuthUserTable: "users",
				},
			},
			ExpectedError: errors.New("API: when 'auth_user_table' is set, 'auth_group_table' must be set as well"),
		}, {
			Name: "api enabled, auth_user_table without db",
			Config: Config{
				API: APIConfig{
					Address:        "0.0.0.0:3000",
					AuthUserTable:  "users",
					AuthGroupTable: "groups",
				},
			},
			ExpectedError: errors.New("API: 'db_type' must be set when 'auth_user_table' is set"),
		}, {
			Name: "api enabled, valid database auth",
			Config: Config{
				API: APIConfig{
					Address:        "0.0.0.0:3000",
					AuthUserTable:  "users",
					AuthGroupTable: "groups",
				},
				Database: DatabaseConfig{
					Type: "sqlite",
				},
			},
		}, {
			Name: "api enabled, valid auth",
			Config: Config{
				API: APIConfig{
					Address: "0.0.0.0:3000",
					Auth:    "abc:def",
				},
			},
		}, {
			Name: "api enabled, valid auth_file",
			Config: Config{
				API: APIConfig{
					Address:  "0.0.0.0:3000",
					AuthFile: "test.json",
				},
			},
		}, {
			Name: "api enabled, jwt should be generated",
			Config: Config{
				API: APIConfig{
					Address: "0.0.0.0:3000",
					Auth:    "abc:def",
				},
			},
			ExpectedJwtSecret: true,
		},
		{
			Name: "api enabled, no key file",
			Config: Config{
				API: APIConfig{
					Address:  "0.0.0.0:3000",
					Auth:     "abc:def",
					CertFile: "/var/lib/rport/server.crt",
					KeyFile:  "",
				},
			},
			ExpectedError: errors.New("API: when 'cert_file' is set, 'key_file' must be set as well"),
		},
		{
			Name: "api enabled, no cert file",
			Config: Config{
				API: APIConfig{
					Address:  "0.0.0.0:3000",
					Auth:     "abc:def",
					CertFile: "",
					KeyFile:  "/var/lib/rport/server.key",
				},
			},
			ExpectedError: errors.New("API: when 'key_file' is set, 'cert_file' must be set as well"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Config.Server = defaultValidMinServerConfig
			err := tc.Config.ParseAndValidate()
			assert.Equal(t, tc.ExpectedError, err)
			if tc.ExpectedJwtSecret {
				assert.NotEmpty(t, tc.Config.API.JWTSecret)
			}
		})
	}
}
