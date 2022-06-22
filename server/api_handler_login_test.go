package chserver

import (
	"context"
	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/share/ptr"
	"github.com/cloudradar-monitoring/rport/share/random"
	"github.com/cloudradar-monitoring/rport/share/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type MockUsersService struct {
	UserService

	ChangeUser     *users.User
	ChangeUsername string
}

func (s *MockUsersService) Change(user *users.User, username string) error {
	s.ChangeUser = user
	s.ChangeUsername = username
	return nil
}

func TestPostToken(t *testing.T) {
	user := &users.User{
		Username: "test-user",
	}
	mockUsersService := &MockUsersService{
		UserService: users.NewAPIService(users.NewStaticProvider([]*users.User{user}), false),
	}

	uuid := "cb5b6578-94f5-4a5b-af58-f7867a943b0c"
	oldUUID := random.UUID4
	random.UUID4 = func() (string, error) {
		return uuid, nil
	}
	defer func() {
		random.UUID4 = oldUUID
	}()

	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			config: &Config{},
		},
		userService: mockUsersService,
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/me/token", nil)
	ctx := api.WithUser(req.Context(), user.Username)
	req = req.WithContext(ctx)
	al.router.ServeHTTP(w, req)

	expectedJSON := `{"data":{"token":"` + uuid + `"}}`
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, expectedJSON, w.Body.String())

	expectedUser := &users.User{
		Token: &uuid,
	}
	assert.Equal(t, user.Username, mockUsersService.ChangeUsername)
	assert.Equal(t, expectedUser, mockUsersService.ChangeUser)
}

func TestDeleteToken(t *testing.T) {
	user := &users.User{
		Username: "test-user",
	}
	mockUsersService := &MockUsersService{
		UserService: users.NewAPIService(users.NewStaticProvider([]*users.User{user}), false),
	}
	noToken := ""
	al := APIListener{
		insecureForTests: true,
		Server: &Server{
			config: &Config{},
		},
		userService: mockUsersService,
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/me/token", nil)
	ctx := api.WithUser(req.Context(), user.Username)
	req = req.WithContext(ctx)
	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	expectedUser := &users.User{
		Token: &noToken,
	}
	assert.Equal(t, user.Username, mockUsersService.ChangeUsername)
	assert.Equal(t, expectedUser, mockUsersService.ChangeUser)
}

func TestWrapWithAuthMiddleware(t *testing.T) {
	ctx := context.Background()

	user := &users.User{
		Username: "user1",
		Password: "$2y$05$ep2DdPDeLDDhwRrED9q/vuVEzRpZtB5WHCFT7YbcmH9r9oNmlsZOm",
		Token:    ptr.String("$2y$05$/D7g/d0sDkNSOh.e6Jzc9OWClcpZ1ieE8Dx.WUaWgayd3Ab0rRdxu"),
	}
	userWithoutToken := &users.User{
		Username: "user2",
		Password: "$2y$05$ep2DdPDeLDDhwRrED9q/vuVEzRpZtB5WHCFT7YbcmH9r9oNmlsZOm",
		Token:    nil,
	}
	al := APIListener{
		apiSessions: newEmptyAPISessionCache(t),
		bannedUsers: security.NewBanList(0),
		userService: users.NewAPIService(users.NewStaticProvider([]*users.User{user, userWithoutToken}), false),
		Server: &Server{
			config: &Config{},
		},
	}
	jwt, err := al.createAuthToken(ctx, time.Hour, user.Username, []Scope{})
	require.NoError(t, err)

	testCases := []struct {
		Name           string
		Username       string
		Password       string
		EnableTwoFA    bool
		Bearer         string
		ExpectedStatus int
	}{
		{
			Name:           "no auth",
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:           "basic auth with password",
			Username:       user.Username,
			Password:       "pwd",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "basic auth with password, no password",
			Username:       user.Username,
			Password:       "",
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:           "basic auth with password, wrong password",
			Username:       user.Username,
			Password:       "wrong",
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:           "basic auth with password, 2fa enabled",
			Username:       user.Username,
			Password:       "pwd",
			EnableTwoFA:    true,
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:           "basic auth with token",
			Username:       user.Username,
			Password:       "token",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "basic auth with token, 2fa enabled",
			Username:       user.Username,
			Password:       "token",
			EnableTwoFA:    true,
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "basic auth with token, wrong token",
			Username:       user.Username,
			Password:       "wrong-token",
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:           "basic auth with token, user has no token",
			Username:       userWithoutToken.Username,
			Password:       "",
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:           "bearer token",
			ExpectedStatus: http.StatusOK,
			Bearer:         jwt,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			twoFATokenDelivery := ""
			if tc.EnableTwoFA {
				twoFATokenDelivery = "smtp"
			}
			al.config.API.TwoFATokenDelivery = twoFATokenDelivery

			handler := al.wrapWithAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, user.Username, api.GetUser(r.Context(), nil))
			}), false)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/some-endpoint", nil)
			if tc.Username != "" {
				req.SetBasicAuth(tc.Username, tc.Password)
			}
			if tc.Bearer != "" {
				req.Header.Set("Authorization", "Bearer "+tc.Bearer)
			}

			handler(w, req)

			assert.Equal(t, tc.ExpectedStatus, w.Code)
		})
	}
}

func TestHandleGetLogin(t *testing.T) {
	authHeader := "Authentication-IsAuthenticated"
	userHeader := "Authentication-User"
	userGroup := "Administrators"
	user := &users.User{
		Username: "user1",
		Password: "$2y$05$ep2DdPDeLDDhwRrED9q/vuVEzRpZtB5WHCFT7YbcmH9r9oNmlsZOm",
		Token:    ptr.String("$2y$05$/D7g/d0sDkNSOh.e6Jzc9OWClcpZ1ieE8Dx.WUaWgayd3Ab0rRdxu"),
	}
	mockUsersService := &MockUsersService{
		UserService: users.NewAPIService(users.NewStaticProvider([]*users.User{user}), false),
	}
	al := APIListener{
		Server: &Server{
			config: &Config{
				API: APIConfig{
					DefaultUserGroup: userGroup,
				},
			},
		},
		bannedUsers: security.NewBanList(0),
		userService: mockUsersService,
		apiSessions: newEmptyAPISessionCache(t),
	}
	al.initRouter()

	testCases := []struct {
		Name              string
		BasicAuthPassword string
		HeaderAuthUser    string
		HeaderAuthEnabled bool
		CreateMissingUser bool
		ExpectedStatus    int
	}{
		{
			Name:           "no auth",
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:              "basic auth",
			BasicAuthPassword: "pwd",
			ExpectedStatus:    http.StatusOK,
		},
		{
			Name:           "header auth - disabled",
			HeaderAuthUser: user.Username,
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:              "header auth - enabled",
			HeaderAuthUser:    user.Username,
			HeaderAuthEnabled: true,
			ExpectedStatus:    http.StatusOK,
		},
		{
			Name:              "header auth + invalid basic auth",
			HeaderAuthUser:    user.Username,
			HeaderAuthEnabled: true,
			BasicAuthPassword: "invalid",
			ExpectedStatus:    http.StatusOK,
		},
		{
			Name:              "header auth - unknown user",
			HeaderAuthUser:    "unknown",
			HeaderAuthEnabled: true,
			CreateMissingUser: true,
			ExpectedStatus:    http.StatusOK,
		}, {
			Name:              "header auth - create missing user",
			HeaderAuthUser:    "new-user",
			HeaderAuthEnabled: true,
			CreateMissingUser: true,
			ExpectedStatus:    http.StatusOK,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			if tc.HeaderAuthEnabled {
				al.config.API.AuthHeader = authHeader
				al.config.API.UserHeader = userHeader
			} else {
				al.config.API.AuthHeader = ""
			}
			al.config.API.CreateMissingUsers = tc.CreateMissingUser

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/login", nil)
			if tc.BasicAuthPassword != "" {
				req.SetBasicAuth(user.Username, tc.BasicAuthPassword)
			}
			if tc.HeaderAuthUser != "" {
				req.Header.Set(authHeader, "1")
				req.Header.Set(userHeader, tc.HeaderAuthUser)
			}

			al.router.ServeHTTP(w, req)

			assert.Equal(t, tc.ExpectedStatus, w.Code)
			if tc.ExpectedStatus == http.StatusOK {
				assert.Contains(t, w.Body.String(), `{"data":{"token":"`)
			}
			if tc.CreateMissingUser {
				assert.Equal(t, tc.HeaderAuthUser, mockUsersService.ChangeUser.Username)
				assert.Equal(t, userGroup, mockUsersService.ChangeUser.Groups[0])
				assert.NotEmpty(t, mockUsersService.ChangeUser.Password)
			}
		})
	}
}
