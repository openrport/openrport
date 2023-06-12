package chserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/share/logger"
)

// TestHandleMeStaticAuth verifies group permissions are not supported
func TestHandleMeStaticAuth(t *testing.T) {
	user := &users.User{
		Username: "test-user",
		Groups:   []string{"group1"},
	}
	userProvider := users.NewStaticProvider([]*users.User{user})
	mockUsersService := &MockUsersService{
		UserService: users.NewAPIService(userProvider, false, 0, -1),
	}
	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			config: &chconfig.Config{},
		},
		userService: mockUsersService,
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	ctx := api.WithUser(req.Context(), user.Username)
	req = req.WithContext(ctx)
	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	expectedJSON := `{
		"data": {
			"username": "test-user",
			"password_expired": false,
			"groups": [
				"group1"
			],
			"two_fa_send_to": "",
			"effective_user_permissions": {
				"auditlog": true,
				"commands": true,
				"monitoring": true,
				"scheduler": true,
				"scripts": true,
				"tunnels": true,
				"uploads": true,
				"vault": true
			},
			"tunnels_restricted":null,
			"commands_restricted":null,
			"group_permissions_enabled": false
		}
	}`
	assert.JSONEq(t, expectedJSON, w.Body.String())
	t.Logf("response %d %s", w.Code, w.Body.String())
}

// TestHandleMeDBAuth verifies group permissions are supported and the user has exemplary access to the vault only.
func TestHandleMeDBAuth(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, err)
	sqlExecs := []string{
		`CREATE TABLE "users" ("username" TEXT PRIMARY KEY, "password" TEXT, "password_expired" BOOLEAN NOT NULL CHECK (password_expired IN (0, 1)) DEFAULT 0)`,
		`INSERT INTO "users" VALUES("test-user","1", false)`,
		`CREATE TABLE "groups" ("username" TEXT, "group" TEXT)`,
		`INSERT INTO "groups" VALUES("test-user","group1")`,
		`CREATE TABLE "group_details" ("name" TEXT, "permissions" TEXT)`,
		`CREATE UNIQUE INDEX "main"."username_group_name" ON "group_details" ("name" ASC)`,
		`INSERT INTO "group_details" VALUES('group1','{"vault":true, "monitoring": true}')`,
	}
	for _, sqlExec := range sqlExecs {
		_, err = db.Exec(sqlExec)
		require.NoError(t, err)
	}

	logfile := t.TempDir() + "/test.log"
	l, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0444)
	require.NoError(t, err, "error creating log file")
	defer l.Close()
	logger := logger.NewLogger("test", logger.LogOutput{File: l}, logger.LogLevelDebug)

	userProvider, err := users.NewUserDatabase(
		db,
		"users",
		"groups",
		"group_details",
		false,
		false,
		false,
		logger)
	require.NoError(t, err)

	mockUsersService := &MockUsersService{
		UserService: users.NewAPIService(userProvider, false, 0, -1),
	}
	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			config: &chconfig.Config{},
		},
		userService: mockUsersService,
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	ctx := api.WithUser(req.Context(), "test-user")
	req = req.WithContext(ctx)
	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	expectedJSON := `{
		"data": {
			"username": "test-user",
			"password_expired": false,
			"groups": [
				"group1"
			],
			"two_fa_send_to": "",
			"effective_user_permissions": {
				"auditlog": false,
				"commands": false,
				"monitoring": true,
				"scheduler": false,
				"scripts": false,
				"tunnels": false,
				"uploads": false,
				"vault": true
			},
			"tunnels_restricted":null,
			"commands_restricted":null,
			"group_permissions_enabled": true
		}
	}`
	assert.JSONEq(t, expectedJSON, w.Body.String())
	t.Logf("response %d %s", w.Code, w.Body.String())
}
