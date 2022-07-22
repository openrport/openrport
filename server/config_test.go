package chserver

import (
	"testing"

	"github.com/cloudradar-monitoring/rport/share/logger"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/api/message"
)

var defaultValidMinServerConfig = ServerConfig{
	URL:          []string{"http://localhost/"},
	DataDir:      "./",
	Auth:         "abc:def",
	UsedPortsRaw: []string{"10-20"},
}
var Mlog = logger.NewMemLogger()

func TestParseAndValidateServerConfig(t *testing.T) {
	testCases := []struct {
		Name          string
		Config        Config
		ExpectedError string
	}{
		{
			Name: "Bad pairing URL",
			Config: Config{
				Server: ServerConfig{
					PairingURL: "ftp:example.com",
					URL:        []string{"http://www.example.com"},
				},
			},
			ExpectedError: "server.pairingURL: invalid url ftp:example.com: schema must be http or https",
		},
		{
			Name: "Bad server connection URL",
			Config: Config{
				Server: ServerConfig{
					PairingURL: "https://pairing.example.com",
					URL:        []string{"https:go.lang"},
				},
			},
			ExpectedError: "server.URL: invalid url https:go.lang, must be absolute url",
		},
		{
			Name: "Bad tunnel host",
			Config: Config{
				Server: ServerConfig{
					URL:          []string{"http://localhost/"},
					DataDir:      "./",
					Auth:         "abc:def",
					UsedPortsRaw: []string{"10-20"},
					TunnelHost:   "bad tunnel host",
				},
			},
			ExpectedError: "invalid tunnel_host 'bad tunnel host': use IP address or FQDN",
		},
		{
			Name: "Correct tunnel host",
			Config: Config{
				Server: ServerConfig{
					URL:          []string{"http://localhost/"},
					DataDir:      "./",
					Auth:         "abc:def",
					UsedPortsRaw: []string{"10-20"},
					TunnelHost:   "tunnel.rport.example.com",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.Config.ParseAndValidate(&Mlog)
			if tc.ExpectedError != "" {
				assert.EqualError(t, err, tc.ExpectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDatabaseParseAndValidate(t *testing.T) {
	testCases := []struct {
		Name           string
		Database       DatabaseConfig
		ExpectedDriver string
		ExpectedDSN    string
		ExpectedError  string
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
			ExpectedError: "invalid 'db_type', expected 'mysql' or 'sqlite', got \"mongodb\"",
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
			if err != nil {
				assert.EqualError(t, err, tc.ExpectedError)
			}
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
		ExpectedError        string
	}{
		{
			Name:          "no auth",
			Config:        Config{},
			ExpectedError: "client authentication must be enabled: set either 'auth', 'auth_file' or 'auth_table'",
		}, {
			Name: "auth and auth_file",
			Config: Config{
				Server: ServerConfig{
					Auth:     "abc:def",
					AuthFile: "test.json",
				},
			},
			ExpectedError: "'auth_file' and 'auth' are both set: expected only one of them",
		}, {
			Name: "auth and auth_table",
			Config: Config{
				Server: ServerConfig{
					Auth:      "abc:def",
					AuthTable: "clients",
				},
			},
			ExpectedError: "'auth' and 'auth_table' are both set: expected only one of them",
		}, {
			Name: "auth_table and auth_file",
			Config: Config{
				Server: ServerConfig{
					AuthTable: "clients",
					AuthFile:  "test.json",
				},
			},
			ExpectedError: "'auth_file' and 'auth_table' are both set: expected only one of them",
		}, {
			Name: "auth_table without db",
			Config: Config{
				Server: ServerConfig{
					AuthTable: "clients",
				},
			},
			ExpectedError: "'db_type' must be set when 'auth_table' is set",
		}, {
			Name: "invalid auth",
			Config: Config{
				Server: ServerConfig{
					Auth: "abc",
				},
			},
			ExpectedError: "invalid client auth credentials, expected '<client-id>:<password>', got \"abc\"",
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
			if tc.ExpectedError != "" {
				assert.EqualError(t, err, tc.ExpectedError, "Error not as expected")
			} else {
				assert.NoError(t, err)
			}
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
		ExpectedError        string
	}{
		{
			Name:   "api disabled, no auth",
			Config: Config{},
		}, {
			Name: "api disabled, doc_root specified",
			Config: Config{
				API: APIConfig{
					DocRoot: "/var/lib/rport/",
				},
			},
			ExpectedError: "API: to use document root you need to specify API address",
		}, {
			Name: "api enabled, no auth",
			Config: Config{
				API: APIConfig{
					Address: "0.0.0.0:3000",
				},
			},
			ExpectedError: "API: authentication must be enabled: set either 'auth', 'auth_file' or 'auth_user_table'",
		}, {
			Name: "api enabled, auth and auth_file",
			Config: Config{
				API: APIConfig{
					Address:  "0.0.0.0:3000",
					Auth:     "abc:def",
					AuthFile: "test.json",
				},
			},
			ExpectedError: "API: 'auth_file' and 'auth' are both set: expected only one of them",
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
			ExpectedError: "API: 'auth_user_table' and 'auth' are both set: expected only one of them",
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
			ExpectedError: "API: 'auth_user_table' and 'auth_file' are both set: expected only one of them",
		}, {
			Name: "api enabled, auth_user_table without auth_group_table",
			Config: Config{
				API: APIConfig{
					Address:       "0.0.0.0:3000",
					AuthUserTable: "users",
				},
			},
			ExpectedError: "API: when 'auth_user_table' is set, 'auth_group_table' must be set as well",
		}, {
			Name: "api enabled, auth_user_table without db",
			Config: Config{
				API: APIConfig{
					Address:        "0.0.0.0:3000",
					AuthUserTable:  "users",
					AuthGroupTable: "groups",
				},
			},
			ExpectedError: "API: 'db_type' must be set when 'auth_user_table' is set",
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
			ExpectedError: "API: when 'cert_file' is set, 'key_file' must be set as well",
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
			ExpectedError: "API: when 'key_file' is set, 'cert_file' must be set as well",
		},
		{
			Name: "api enabled, single user auth, 2fa enabled",
			Config: Config{
				API: APIConfig{
					Address:            "0.0.0.0:3000",
					Auth:               "abc:def",
					TwoFATokenDelivery: "/bin/sh",
				},
			},
			ExpectedError: "API: 2FA is not available if you use a single static user-password pair",
		},
		{
			Name: "api enabled, unknown 2fa method",
			Config: Config{
				API: APIConfig{
					Address:            "0.0.0.0:3000",
					AuthFile:           "test.json",
					TwoFATokenDelivery: "unknown",
				},
			},
			ExpectedError: "API: unknown 2fa token delivery method: unknown",
		},
		{
			Name: "api enabled, script 2fa method, invalid send to type",
			Config: Config{
				API: APIConfig{
					Address:            "0.0.0.0:3000",
					AuthFile:           "test.json",
					TwoFATokenDelivery: "/bin/sh",
					TwoFASendToType:    "invalid",
				},
			},
			ExpectedError: `API: invalid api.two_fa_send_to_type: "invalid"`,
		},
		{
			Name: "api enabled, script 2fa method, invalid send to regex",
			Config: Config{
				API: APIConfig{
					Address:            "0.0.0.0:3000",
					AuthFile:           "test.json",
					TwoFATokenDelivery: "/bin/sh",
					TwoFASendToType:    message.ValidationRegex,
					TwoFASendToRegex:   "[a-z",
				},
			},
			ExpectedError: "API: invalid api.two_fa_send_to_regex: error parsing regexp: missing closing ]: `[a-z`",
		},
		{
			Name: "api enabled, script 2fa method, ok",
			Config: Config{
				API: APIConfig{
					Address:            "0.0.0.0:3000",
					AuthFile:           "test.json",
					TwoFATokenDelivery: "/bin/sh",
					TwoFASendToType:    message.ValidationRegex,
					TwoFASendToRegex:   "[a-z]{10}",
				},
			},
		},
		{
			Name: "api enabled, auth_header no user_header",
			Config: Config{
				API: APIConfig{
					Address:    "0.0.0.0:3000",
					AuthHeader: "Authentication-IsAuthenticated",
					AuthFile:   "test.json",
				},
			},
			ExpectedError: "API: 'user_header' must be set when 'auth_header' is set",
		},
		{
			Name: "api enabled, auth_header with auth",
			Config: Config{
				API: APIConfig{
					Address:    "0.0.0.0:3000",
					AuthHeader: "Authentication-IsAuthenticated",
					UserHeader: "Authentication-User",
					Auth:       "abc:def",
				},
			},
			ExpectedError: "API: 'auth_header' cannot be used with single user 'auth'",
		},
		{
			Name: "api enabled, auth_header ok",
			Config: Config{
				API: APIConfig{
					Address:    "0.0.0.0:3000",
					AuthHeader: "Authentication-IsAuthenticated",
					UserHeader: "Authentication-User",
					AuthFile:   "test.json",
				},
			},
		},
		{
			Name: "totp enabled ok",
			Config: Config{
				API: APIConfig{
					Address:     "0.0.0.0:3000",
					AuthFile:    "test.json",
					TotPEnabled: true,
				},
			},
		},
		{
			Name: "totp enabled, 2fa enabled, conflict",
			Config: Config{
				API: APIConfig{
					Address:            "0.0.0.0:3000",
					AuthFile:           "test.json",
					TotPEnabled:        true,
					TwoFATokenDelivery: "/bin/sh",
					TwoFASendToType:    message.ValidationRegex,
				},
			},
			ExpectedError: "API: conflicting 2FA configuration, two factor auth and totp_enabled options cannot be both enabled",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Config.Server = defaultValidMinServerConfig
			err := tc.Config.ParseAndValidate(&Mlog)
			if tc.ExpectedError != "" {
				assert.EqualError(t, err, tc.ExpectedError)
			} else {
				assert.NoError(t, err)
			}
			if tc.ExpectedJwtSecret {
				assert.NotEmpty(t, tc.Config.API.JWTSecret)
			}
		})
	}
}

func TestParseAndValidatePorts(t *testing.T) {
	testCases := []struct {
		Name                      string
		Config                    ServerConfig
		ExpectedAllowedPorts      mapset.Set
		ExpectedAllowedPortsCount int
		ExpectedErrorStr          string
	}{
		{
			Name: "default values",
			Config: ServerConfig{
				UsedPortsRaw:     []string{"20000-30000"},
				ExcludedPortsRaw: []string{"1-1024"},
			},
			ExpectedAllowedPortsCount: 10001,
		},
		{
			Name: "excluded ports ignored",
			Config: ServerConfig{
				UsedPortsRaw:     []string{"45-50"},
				ExcludedPortsRaw: []string{"1-10", "44", "51", "80-90"},
			},
			ExpectedAllowedPorts: mapset.NewThreadUnsafeSetFromSlice([]interface{}{45, 46, 47, 48, 49, 50}),
		},
		{
			Name: "used ports and excluded ports",
			Config: ServerConfig{
				UsedPortsRaw:     []string{"100-200", "205", "250-300", "305", "400-500"},
				ExcludedPortsRaw: []string{"80-110", "114-116", "118", "120-198", "200", "240-310", "305", "401-499"},
			},
			ExpectedAllowedPorts: mapset.NewThreadUnsafeSetFromSlice([]interface{}{111, 112, 113, 117, 119, 199, 205, 400, 500}),
		},
		{
			Name: "excluded ports empty",
			Config: ServerConfig{
				UsedPortsRaw:     []string{"45-46"},
				ExcludedPortsRaw: []string{},
			},
			ExpectedAllowedPorts: mapset.NewThreadUnsafeSetFromSlice([]interface{}{45, 46}),
		},
		{
			Name: "one allowed port",
			Config: ServerConfig{
				UsedPortsRaw:     []string{"20000"},
				ExcludedPortsRaw: []string{},
			},
			ExpectedAllowedPorts: mapset.NewThreadUnsafeSetFromSlice([]interface{}{20000}),
		},
		{
			Name: "both empty",
			Config: ServerConfig{
				UsedPortsRaw:     []string{},
				ExcludedPortsRaw: []string{},
			},
			ExpectedErrorStr: "invalid 'used_ports', 'excluded_ports': at least one port should be available for port assignment",
		},
		{
			Name: "invalid used ports",
			Config: ServerConfig{
				UsedPortsRaw:     []string{"9999999999"},
				ExcludedPortsRaw: []string{"1-1024"},
			},
			ExpectedErrorStr: "can't parse 'used_ports': invalid port number: 9999999999",
		},
		{
			Name: "invalid excluded ports",
			Config: ServerConfig{
				UsedPortsRaw:     []string{"10-20"},
				ExcludedPortsRaw: []string{"a"},
			},
			ExpectedErrorStr: `can't parse 'excluded_ports': can't parse port number a: strconv.Atoi: parsing "a": invalid syntax`,
		},
		{
			Name: "no available allowed ports",
			Config: ServerConfig{
				UsedPortsRaw:     []string{"1-1024", "20000-30000"},
				ExcludedPortsRaw: []string{"1-1024", "20000-25000", "25001-29999", "30000"},
			},
			ExpectedErrorStr: "invalid 'used_ports', 'excluded_ports': at least one port should be available for port assignment",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			actualErr := tc.Config.parseAndValidatePorts()
			if tc.ExpectedErrorStr != "" {
				require.EqualError(t, actualErr, tc.ExpectedErrorStr)
			} else {
				require.NoError(t, actualErr)
				if tc.ExpectedAllowedPorts != nil {
					assert.EqualValues(t, tc.ExpectedAllowedPorts, tc.Config.allowedPorts)
				} else {
					assert.Equal(t, tc.ExpectedAllowedPortsCount, tc.Config.allowedPorts.Cardinality())
				}
			}
		})
	}
}
