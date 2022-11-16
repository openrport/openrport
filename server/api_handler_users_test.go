package chserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/session"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/bearer"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/security"
)

type UserAPISessionsResponse struct {
	Data []*session.APISession
}

func TestShouldHandleGetAllUserAPISessions(t *testing.T) {
	al, adminUser := setupTestAPIListenerUserAPISessions(t, nil)

	ctx := context.Background()
	testRunTime := time.Now()

	adminUserJWT, err := bearer.CreateAuthToken(ctx, al.apiSessions, al.config.API.JWTSecret, time.Hour, adminUser.Username, []bearer.Scope{}, "1.2.3.4", "Safari")
	require.NoError(t, err)

	w := httptest.NewRecorder()

	req := httptest.NewRequest("GET", "/api/v1/users/admin/sessions", nil)
	req.SetBasicAuth("admin", "foobaz")

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	sessionsResponse := &UserAPISessionsResponse{}

	err = json.Unmarshal(w.Body.Bytes(), sessionsResponse)
	require.NoError(t, err)

	adminSession := sessionsResponse.Data[0]

	assert.Equal(t, adminUserJWT, adminSession.Token)
	assert.Equal(t, adminUser.Username, adminSession.Username)
	assert.Less(t, testRunTime, adminSession.LastAccessAt)
}

func TestShouldHandleDeleteUserSession(t *testing.T) {
	al, adminUser := setupTestAPIListenerUserAPISessions(t, nil)

	ctx := context.Background()

	_, err := bearer.CreateAuthToken(ctx, al.apiSessions, al.config.API.JWTSecret, time.Hour, adminUser.Username, []bearer.Scope{}, "1.2.3.4", "Safari")
	require.NoError(t, err)

	w := httptest.NewRecorder()

	req := httptest.NewRequest("DELETE", "/api/v1/users/admin/sessions/1", nil)
	req.SetBasicAuth("admin", "foobaz")

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Result().StatusCode)
}

func TestShouldHandleDeleteAllUserSessions(t *testing.T) {
	al, adminUser := setupTestAPIListenerUserAPISessions(t, nil)

	ctx := context.Background()

	_, err := bearer.CreateAuthToken(ctx, al.apiSessions, al.config.API.JWTSecret, time.Hour, adminUser.Username, []bearer.Scope{}, "1.2.3.4", "Safari")
	require.NoError(t, err)

	w := httptest.NewRecorder()

	req := httptest.NewRequest("DELETE", "/api/v1/users/admin/sessions", nil)
	req.SetBasicAuth("admin", "foobaz")

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Result().StatusCode)
}

func TestShouldHandleAuthorization(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		method     string
		path       string
		withAuth   bool
		statusCode int
	}{
		{
			name:       "get user sessions, authorized",
			method:     "GET",
			path:       "users/admin/sessions",
			withAuth:   true,
			statusCode: http.StatusOK,
		},
		{
			name:       "get user sessions, unauthorized",
			method:     "GET",
			path:       "users/admin/sessions",
			withAuth:   false,
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "delete user session, authorized",
			method:     "DELETE",
			path:       "users/admin/sessions/1",
			withAuth:   true,
			statusCode: http.StatusNoContent,
		},
		{
			name:       "delete user session, unauthorized",
			method:     "DELETE",
			path:       "users/admin/sessions/1",
			withAuth:   false,
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "delete user sessions, authorized",
			method:     "DELETE",
			path:       "users/admin/sessions",
			withAuth:   true,
			statusCode: http.StatusNoContent,
		},
		{
			name:       "delete user sessions, unauthorized",
			method:     "DELETE",
			path:       "users/admin/sessions",
			withAuth:   false,
			statusCode: http.StatusUnauthorized,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			al, adminUser := setupTestAPIListenerUserAPISessions(t, nil)

			_, err := bearer.CreateAuthToken(ctx, al.apiSessions, al.config.API.JWTSecret, time.Hour, adminUser.Username, []bearer.Scope{}, "1.2.3.4", "Safari")
			require.NoError(t, err)

			w := httptest.NewRecorder()

			req := httptest.NewRequest(tc.method, "/api/v1/"+tc.path, nil)
			if tc.withAuth {
				req.SetBasicAuth("admin", "foobaz")
			}

			al.router.ServeHTTP(w, req)

			assert.Equal(t, tc.statusCode, w.Result().StatusCode)
		})
	}
}

func TestShouldErrorWhenNonAdminUser(t *testing.T) {
	al, adminUser := setupTestAPIListenerUserAPISessions(t, nil)

	ctx := context.Background()

	nonAdminUser := &users.User{
		Username: "user1",
		Password: "pa55word",
		Groups:   []string{},
	}

	al.userService = users.NewAPIService(users.NewStaticProvider([]*users.User{adminUser, nonAdminUser}), false)

	_, err := bearer.CreateAuthToken(ctx, al.apiSessions, al.config.API.JWTSecret, time.Hour, adminUser.Username, []bearer.Scope{}, "1.2.3.4", "Safari")
	require.NoError(t, err)

	_, err = bearer.CreateAuthToken(ctx, al.apiSessions, al.config.API.JWTSecret, time.Hour, nonAdminUser.Username, []bearer.Scope{}, "1.2.3.4", "Safari")
	require.NoError(t, err)

	w := httptest.NewRecorder()

	req := httptest.NewRequest("GET", "/api/v1/users/user1/sessions", nil)
	req.SetBasicAuth(nonAdminUser.Username, nonAdminUser.Password)

	al.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode)
}

type MockAPISessionStorageProvider struct {
	*session.SqliteProvider

	shouldFailDeleteAllByUser bool
	shouldFailDeleteByUser    bool
}

func newEmptyMockAPISessionStorageProvider() (m *MockAPISessionStorageProvider, err error) {
	m = &MockAPISessionStorageProvider{}
	m.SqliteProvider, err = session.NewSqliteProvider(":memory:", DataSourceOptions)
	return m, err
}

func (p *MockAPISessionStorageProvider) DeleteAllByUser(ctx context.Context, username string) (err error) {
	if p.shouldFailDeleteAllByUser {
		return errors.New("a random error")
	}
	return p.SqliteProvider.DeleteAllByUser(ctx, username)
}

func (p *MockAPISessionStorageProvider) DeleteByUser(ctx context.Context, username string, sessionID int64) (err error) {
	if p.shouldFailDeleteByUser {
		return errors.New("another random error")
	}
	return p.SqliteProvider.DeleteByUser(ctx, username, sessionID)
}

type MockAPISessionCacheProvider struct {
	*cache.Cache

	shouldFailGet bool
}

func newEmptyMockAPISessionCacheProvider(defaultExpiration, cleanupInterval time.Duration) (m *MockAPISessionCacheProvider) {
	m = &MockAPISessionCacheProvider{}
	m.Cache = cache.New(defaultExpiration, cleanupInterval)
	return m
}

func (mp *MockAPISessionCacheProvider) Items() map[string]cache.Item {
	if mp.shouldFailGet {
		badItems := map[string]cache.Item{
			"0": {
				Object: "bad object",
			},
		}
		return badItems
	}
	return mp.Cache.Items()
}

func TestShouldErrorIfUnableToGetAllSessionsForUser(t *testing.T) {
	cp := newEmptyMockAPISessionCacheProvider(bearer.DefaultTokenLifetime, cleanupAPISessionsInterval)
	cp.shouldFailGet = true

	mp, err := newEmptyMockAPISessionStorageProvider()
	require.NoError(t, err)

	sessionCache := newEmptyAPISessionCacheWithProviders(t, mp, cp)

	al, adminUser := setupTestAPIListenerUserAPISessions(t, sessionCache)

	ctx := context.Background()

	_, err = bearer.CreateAuthToken(ctx, al.apiSessions, al.config.API.JWTSecret, time.Hour, adminUser.Username, []bearer.Scope{}, "1.2.3.4", "Safari")
	require.NoError(t, err)

	w := httptest.NewRecorder()

	req := httptest.NewRequest("GET", "/api/v1/users/admin/sessions", nil)
	req.SetBasicAuth("admin", "foobaz")

	al.router.ServeHTTP(w, req)

	errInfo := &api.ErrorPayload{}
	err = json.Unmarshal(w.Body.Bytes(), &errInfo)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Equal(t, "unable to get sessions for user \"admin\"", errInfo.Errors[0].Title)
	assert.Equal(t, "invalid cache entry: expected *APISession, got string", errInfo.Errors[0].Detail)
}

func TestShouldErrorIfUnableToDeleteSessionForUser(t *testing.T) {
	cp := newEmptyMockAPISessionCacheProvider(bearer.DefaultTokenLifetime, cleanupAPISessionsInterval)

	mp, err := newEmptyMockAPISessionStorageProvider()
	require.NoError(t, err)
	mp.shouldFailDeleteByUser = true

	sessionCache := newEmptyAPISessionCacheWithProviders(t, mp, cp)

	al, adminUser := setupTestAPIListenerUserAPISessions(t, sessionCache)

	ctx := context.Background()

	_, err = bearer.CreateAuthToken(ctx, al.apiSessions, al.config.API.JWTSecret, time.Hour, adminUser.Username, []bearer.Scope{}, "1.2.3.4", "Safari")
	require.NoError(t, err)

	w := httptest.NewRecorder()

	req := httptest.NewRequest("DELETE", "/api/v1/users/admin/sessions/1", nil)
	req.SetBasicAuth("admin", "foobaz")

	al.router.ServeHTTP(w, req)

	errInfo := &api.ErrorPayload{}
	err = json.Unmarshal(w.Body.Bytes(), &errInfo)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Equal(t, "unable to delete session \"1\" for user \"admin\"", errInfo.Errors[0].Title)
	assert.Equal(t, "unable to delete session from cache: another random error", errInfo.Errors[0].Detail)
}

func TestShouldErrorIfUnableToDeleteAllSessionsForUser(t *testing.T) {
	cp := newEmptyMockAPISessionCacheProvider(bearer.DefaultTokenLifetime, cleanupAPISessionsInterval)

	mp, err := newEmptyMockAPISessionStorageProvider()
	require.NoError(t, err)
	mp.shouldFailDeleteAllByUser = true

	sessionCache := newEmptyAPISessionCacheWithProviders(t, mp, cp)

	al, adminUser := setupTestAPIListenerUserAPISessions(t, sessionCache)

	ctx := context.Background()

	_, err = bearer.CreateAuthToken(ctx, al.apiSessions, al.config.API.JWTSecret, time.Hour, adminUser.Username, []bearer.Scope{}, "1.2.3.4", "Safari")
	require.NoError(t, err)

	w := httptest.NewRecorder()

	req := httptest.NewRequest("DELETE", "/api/v1/users/admin/sessions", nil)
	req.SetBasicAuth("admin", "foobaz")

	al.router.ServeHTTP(w, req)

	errInfo := &api.ErrorPayload{}
	err = json.Unmarshal(w.Body.Bytes(), &errInfo)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Equal(t, "unable to delete all sessions for user \"admin\"", errInfo.Errors[0].Title)
	assert.Equal(t, "unable to delete sessions from cache: a random error", errInfo.Errors[0].Detail)
}

func newEmptyAPISessionCacheWithProviders(t *testing.T, mp *MockAPISessionStorageProvider, cp *MockAPISessionCacheProvider) *session.Cache {
	c, err := session.NewCache(context.Background(), time.Hour, time.Hour, mp, cp)
	require.NoError(t, err)
	return c
}

func setupTestAPIListenerUserAPISessions(t *testing.T, sessionCache *session.Cache) (al *APIListener, adminUser *users.User) {
	t.Helper()

	testlog := logger.NewLogger("test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	adminUser = &users.User{
		Username: "admin",
		Password: "foobaz",
		Groups:   []string{users.Administrators},
	}

	serverCfg := &Config{
		API: APIConfig{
			Auth: "admin:foobaz",
		},
	}

	if sessionCache == nil {
		sessionCache = newEmptyAPISessionCache(t)
	}

	al = &APIListener{
		Logger: testlog,
		Server: &Server{
			config: serverCfg,
		},
		bannedUsers: security.NewBanList(0),
		apiSessions: sessionCache,
		userService: users.NewAPIService(users.NewStaticProvider([]*users.User{adminUser}), false),
	}
	al.initRouter()

	return al, adminUser
}
