package chserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	"github.com/cloudradar-monitoring/rport/server/test/jb"
	"github.com/cloudradar-monitoring/rport/server/test/sb"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"
)

var testLog = chshare.NewLogger("api-listener-test", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)
var hour = time.Hour

type JobProviderMock struct {
	ReturnJob          *models.Job
	ReturnJobSummaries []*models.JobSummary
	ReturnErr          error

	InputSID string
	InputJID string
	InputJob *models.Job
}

func NewJobProviderMock() *JobProviderMock {
	return &JobProviderMock{}
}

func (p *JobProviderMock) GetByJID(sid, jid string) (*models.Job, error) {
	p.InputSID = sid
	p.InputJID = jid
	return p.ReturnJob, p.ReturnErr
}

func (p *JobProviderMock) GetSummariesBySID(sid string) ([]*models.JobSummary, error) {
	p.InputSID = sid
	return p.ReturnJobSummaries, p.ReturnErr
}

func (p *JobProviderMock) SaveJob(job *models.Job) error {
	p.InputJob = job
	return p.ReturnErr
}

func TestGetCorrespondingSortFuncPositive(t *testing.T) {
	testCases := []struct {
		sortStr string

		wantFunc func(a []*sessions.ClientSession, desc bool)
		wantDesc bool
	}{
		{
			sortStr:  "",
			wantFunc: sessions.SortByID,
			wantDesc: false,
		},
		{
			sortStr:  "-",
			wantFunc: sessions.SortByID,
			wantDesc: true,
		},
		{
			sortStr:  "id",
			wantFunc: sessions.SortByID,
			wantDesc: false,
		},
		{
			sortStr:  "-id",
			wantFunc: sessions.SortByID,
			wantDesc: true,
		},
		{
			sortStr:  "name",
			wantFunc: sessions.SortByName,
			wantDesc: false,
		},
		{
			sortStr:  "-name",
			wantFunc: sessions.SortByName,
			wantDesc: true,
		},
		{
			sortStr:  "hostname",
			wantFunc: sessions.SortByHostname,
			wantDesc: false,
		},
		{
			sortStr:  "-hostname",
			wantFunc: sessions.SortByHostname,
			wantDesc: true,
		},
		{
			sortStr:  "os",
			wantFunc: sessions.SortByOS,
			wantDesc: false,
		},
		{
			sortStr:  "-os",
			wantFunc: sessions.SortByOS,
			wantDesc: true,
		},
	}

	for _, tc := range testCases {
		// when
		gotFunc, gotDesc, gotErr := getCorrespondingSortFunc(tc.sortStr)

		// then
		// workaround to compare func vars, see https://github.com/stretchr/testify/issues/182
		wantFuncName := runtime.FuncForPC(reflect.ValueOf(tc.wantFunc).Pointer()).Name()
		gotFuncName := runtime.FuncForPC(reflect.ValueOf(gotFunc).Pointer()).Name()
		msg := fmt.Sprintf("getCorrespondingSortFunc(%q) = (%s, %v, %v), expected: (%s, %v, %v)", tc.sortStr, gotFuncName, gotDesc, gotErr, wantFuncName, tc.wantDesc, nil)

		assert.NoErrorf(t, gotErr, msg)
		assert.Equalf(t, wantFuncName, gotFuncName, msg)
		assert.Equalf(t, tc.wantDesc, gotDesc, msg)
	}
}

func TestGetCorrespondingSortFuncNegative(t *testing.T) {
	// when
	_, _, gotErr := getCorrespondingSortFunc("unknown")

	// then
	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "incorrect format")
}

var (
	cl1 = &clients.Client{ID: "user1", Password: "pswd1"}
	cl2 = &clients.Client{ID: "user2", Password: "pswd2"}
	cl3 = &clients.Client{ID: "user3", Password: "pswd3"}

	clientsFileProvider  = clients.NewFileClients(testLog, "test_file")
	singleClientProvider = clients.NewSingleClient(testLog, cl1.ID+":"+cl1.Password)
)

func TestHandleGetClients(t *testing.T) {
	require := require.New(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)

	testClients := clients.NewClientCache([]*clients.Client{cl1, cl3, cl2})
	testCases := []struct {
		descr string // Test Case Description

		clientProvider ClientProvider
		clientCache    *clients.ClientCache

		wantStatusCode int
		wantClients    []*clients.Client
		wantErrCode    string
		wantErrTitle   string
	}{
		{
			descr:          "auth file, 3 clients",
			clientProvider: clientsFileProvider,
			clientCache:    testClients,
			wantStatusCode: http.StatusOK,
			wantClients:    []*clients.Client{cl1, cl2, cl3},
		},
		{
			descr:          "auth file, no clients",
			clientProvider: clientsFileProvider,
			clientCache:    clients.NewEmptyClientCache(),
			wantStatusCode: http.StatusOK,
			wantClients:    []*clients.Client{},
		},
		{
			descr:          "auth, single client",
			clientProvider: singleClientProvider,
			clientCache:    clients.NewClientCache([]*clients.Client{cl1}),
			wantStatusCode: http.StatusOK,
			wantClients:    []*clients.Client{cl1},
		},
		{
			descr:          "auth file, unset clients cache",
			clientProvider: clientsFileProvider,
			clientCache:    nil,
			wantStatusCode: http.StatusInternalServerError,
			wantErrTitle:   "Rport clients cache is not initialized.",
		},
		{
			descr:          "no auth",
			clientProvider: nil,
			clientCache:    clients.NewEmptyClientCache(),
			wantStatusCode: http.StatusMethodNotAllowed,
			wantErrTitle:   "Client authentication is disabled.",
			wantErrCode:    ErrCodeClientAuthDisabled,
		},
	}

	for _, tc := range testCases {
		msg := fmt.Sprintf("test case: %q", tc.descr)

		// given
		al := APIListener{
			Logger:         testLog,
			clientProvider: tc.clientProvider,
			Server: &Server{
				clientCache: tc.clientCache,
				config: &Config{
					Server: ServerConfig{MaxRequestBytes: 1024 * 1024},
				},
			},
		}

		// when
		handler := http.HandlerFunc(al.handleGetClients)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// then
		require.Equalf(tc.wantStatusCode, w.Code, msg)
		var wantResp interface{}
		if tc.wantErrTitle == "" {
			// success case
			wantResp = api.NewSuccessPayload(tc.wantClients)
		} else {
			// failure case
			wantResp = api.NewErrorPayloadWithCode(tc.wantErrCode, tc.wantErrTitle, "")
		}
		wantRespBytes, err := json.Marshal(wantResp)
		require.NoErrorf(err, msg)
		require.Equalf(string(wantRespBytes), w.Body.String(), msg)
	}
}

func TestHandlePostClients(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	composeRequestBody := func(id, pswd string) io.Reader {
		c := clients.Client{ID: id, Password: pswd}
		b, err := json.Marshal(c)
		require.NoError(err)
		return bytes.NewBuffer(b)
	}
	cl4 := &clients.Client{ID: "user4", Password: "pswd4"}
	initCacheState := []*clients.Client{cl1, cl2, cl3}

	testCases := []struct {
		descr string // Test Case Description

		clientProvider  ClientProvider
		clientCache     *clients.ClientCache
		clientAuthWrite bool
		requestBody     io.Reader

		wantStatusCode int
		wantClients    []*clients.Client
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
	}{
		{
			descr:           "auth file, new valid client",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, cl4.Password),
			wantStatusCode:  http.StatusCreated,
			wantClients:     []*clients.Client{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, new valid client, empty cache",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewEmptyClientCache(),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, cl4.Password),
			wantStatusCode:  http.StatusCreated,
			wantClients:     []*clients.Client{cl4},
		},
		{
			descr:           "auth file, new valid client, clients cache is not initialized",
			clientProvider:  clientsFileProvider,
			clientCache:     nil,
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, cl4.Password),
			wantStatusCode:  http.StatusInternalServerError,
			wantErrTitle:    "Rport clients cache is not initialized.",
		},
		{
			descr:           "auth file, empty request body",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     strings.NewReader(""),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Missing data.",
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, invalid request body",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     strings.NewReader("invalid json"),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid JSON data.",
			wantErrDetail:   "invalid character 'i' looking for beginning of value",
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, invalid request, empty id",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     composeRequestBody("", cl4.Password),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing ID.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, invalid request, 'id' is missing",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     strings.NewReader(`{"password":"pswd"}`),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing ID.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, invalid request, empty password",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, ""),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing password.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, invalid request, 'password' is missing",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     strings.NewReader(`{"id":"user"}`),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing password.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, invalid request, id too short",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     composeRequestBody("12", cl4.Password),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing ID.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, invalid request, password too short",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, "12"),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing password.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, client already exist",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl1.ID, cl4.Password),
			wantStatusCode:  http.StatusConflict,
			wantErrCode:     ErrCodeAlreadyExist,
			wantErrTitle:    fmt.Sprintf("Client with ID %q already exist.", cl1.ID),
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, auth in Read-Only mode",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: false,
			requestBody:     composeRequestBody(cl1.ID, cl4.Password),
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthRO,
			wantErrTitle:    "Client authentication has been attached in read-only mode.",
			wantClients:     initCacheState,
		},
		{
			descr:           "auth, single client",
			clientProvider:  singleClientProvider,
			clientCache:     clients.NewClientCache([]*clients.Client{cl1}),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, cl4.Password),
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthSingleClient,
			wantErrTitle:    "Client authentication is enabled only for a single user.",
			wantClients:     []*clients.Client{cl1},
		},
		{
			descr:           "no auth",
			clientProvider:  nil,
			clientCache:     nil,
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, cl4.Password),
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthDisabled,
			wantErrTitle:    "Client authentication is disabled.",
		},
	}

	for _, tc := range testCases {
		msg := fmt.Sprintf("test case: %q", tc.descr)

		// given
		al := APIListener{
			Server: &Server{
				clientCache: tc.clientCache,
				config: &Config{
					Server: ServerConfig{
						AuthWrite:       tc.clientAuthWrite,
						MaxRequestBytes: 1024 * 1024,
					},
				},
			},
			Logger:         testLog,
			clientProvider: tc.clientProvider,
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", tc.requestBody)

		// when
		handler := http.HandlerFunc(al.handlePostClients)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// then
		require.Equalf(tc.wantStatusCode, w.Code, msg)
		if tc.wantErrTitle == "" {
			// success case
			assert.Emptyf(w.Body.String(), msg)
		} else {
			// failure case
			wantResp := api.NewErrorPayloadWithCode(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
			wantRespBytes, err := json.Marshal(wantResp)
			require.NoErrorf(err, msg)
			require.Equalf(string(wantRespBytes), w.Body.String(), msg)
		}
		if al.clientCache != nil {
			assert.ElementsMatchf(tc.wantClients, al.clientCache.GetAll(), msg)
		}
	}
}

type mockConnection struct {
	ssh.Conn
	closed bool
}

func (m *mockConnection) Close() error {
	m.closed = true
	return nil
}

func TestHandleDeleteClient(t *testing.T) {
	mockConn := &mockConnection{}

	initCacheState := []*clients.Client{cl1, cl2, cl3}

	s1 := sb.New(t).ClientID(&cl1.ID).Connection(mockConn).Build()
	s2 := sb.New(t).ClientID(&cl1.ID).DisconnectedDuration(5 * time.Minute).Build()

	testCases := []struct {
		descr string // Test Case Description

		clientProvider  ClientProvider
		clientCache     *clients.ClientCache
		sessions        []*sessions.ClientSession
		clientAuthWrite bool
		clientID        string
		urlSuffix       string

		wantStatusCode int
		wantClients    []*clients.Client
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
		wantClosedConn bool
		wantSessions   []*sessions.ClientSession
	}{
		{
			descr:           "auth file, success delete",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			clientID:        cl1.ID,
			wantStatusCode:  http.StatusNoContent,
			wantClients:     []*clients.Client{cl2, cl3},
		},
		{
			descr:           "auth file, clients cache is not initialized",
			clientProvider:  clientsFileProvider,
			clientCache:     nil,
			clientAuthWrite: true,
			clientID:        cl1.ID,
			wantStatusCode:  http.StatusInternalServerError,
			wantErrTitle:    "Rport clients cache is not initialized.",
		},
		{
			descr:           "auth file, missing client ID",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: true,
			clientID:        "unknown-client-id",
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeClientNotFound,
			wantErrTitle:    fmt.Sprintf("Client with ID=%q not found.", "unknown-client-id"),
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, client has active session",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			sessions:        []*sessions.ClientSession{s1},
			clientAuthWrite: true,
			clientID:        cl1.ID,
			wantStatusCode:  http.StatusConflict,
			wantErrCode:     ErrCodeClientHasSession,
			wantErrTitle:    fmt.Sprintf("Client expected to have no active or disconnected session(s), got %d.", 1),
			wantClients:     initCacheState,
			wantSessions:    []*sessions.ClientSession{s1},
		},
		{
			descr:           "auth file, client has disconnected session",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			sessions:        []*sessions.ClientSession{s2},
			clientAuthWrite: true,
			clientID:        cl1.ID,
			wantStatusCode:  http.StatusConflict,
			wantErrCode:     ErrCodeClientHasSession,
			wantErrTitle:    fmt.Sprintf("Client expected to have no active or disconnected session(s), got %d.", 1),
			wantClients:     initCacheState,
			wantSessions:    []*sessions.ClientSession{s2},
		},
		{
			descr:           "auth file, auth in Read-Only mode",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			clientAuthWrite: false,
			clientID:        cl1.ID,
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthRO,
			wantErrTitle:    "Client authentication has been attached in read-only mode.",
			wantClients:     initCacheState,
		},
		{
			descr:           "auth file, client has active session, force",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			sessions:        []*sessions.ClientSession{s1},
			clientAuthWrite: true,
			clientID:        cl1.ID,
			urlSuffix:       "?force=true",
			wantStatusCode:  http.StatusNoContent,
			wantClients:     []*clients.Client{cl2, cl3},
			wantClosedConn:  true,
		},
		{
			descr:           "auth file, client has disconnected session, force",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			sessions:        []*sessions.ClientSession{s2},
			clientAuthWrite: true,
			clientID:        cl1.ID,
			urlSuffix:       "?force=true",
			wantStatusCode:  http.StatusNoContent,
			wantClients:     []*clients.Client{cl2, cl3},
		},
		{
			descr:           "invalid force param",
			clientProvider:  clientsFileProvider,
			clientCache:     clients.NewClientCache(initCacheState),
			sessions:        []*sessions.ClientSession{s1, s2},
			clientAuthWrite: true,
			clientID:        cl1.ID,
			urlSuffix:       "?force=test",
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid force param test.",
			wantClients:     initCacheState,
			wantSessions:    []*sessions.ClientSession{s1, s2},
		},
		{
			descr:           "auth, single client",
			clientProvider:  singleClientProvider,
			clientCache:     clients.NewClientCache([]*clients.Client{cl1}),
			clientAuthWrite: true,
			clientID:        cl1.ID,
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthSingleClient,
			wantErrTitle:    "Client authentication is enabled only for a single user.",
			wantClients:     []*clients.Client{cl1},
		},
		{
			descr:           "no auth",
			clientProvider:  nil,
			clientCache:     nil,
			clientAuthWrite: true,
			clientID:        cl1.ID,
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthDisabled,
			wantErrTitle:    "Client authentication is disabled.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.descr, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			// given
			al := APIListener{
				Server: &Server{
					sessionService: NewSessionService(nil, sessions.NewSessionRepository(tc.sessions, &hour)),
					clientCache:    tc.clientCache,
					config: &Config{
						Server: ServerConfig{
							AuthWrite:       tc.clientAuthWrite,
							MaxRequestBytes: 1024 * 1024,
						},
					},
				},
				Logger:         testLog,
				clientProvider: tc.clientProvider,
			}
			al.initRouter()
			mockConn.closed = false

			url := fmt.Sprintf("/api/v1/clients/%s", tc.clientID)
			url += tc.urlSuffix
			req := httptest.NewRequest(http.MethodDelete, url, nil)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(tc.wantStatusCode, w.Code)
			var wantRespStr string
			if tc.wantErrTitle == "" {
				// success case: empty body
			} else {
				// failure case
				wantResp := api.NewErrorPayloadWithCode(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(err)
				wantRespStr = string(wantRespBytes)
			}
			assert.Equal(wantRespStr, w.Body.String())
			if al.clientCache != nil {
				assert.ElementsMatch(tc.wantClients, al.clientCache.GetAll())
			}
			assert.Equal(tc.wantClosedConn, mockConn.closed)
			allSessions, err := al.sessionService.GetAll()
			require.NoError(err)
			assert.ElementsMatch(tc.wantSessions, allSessions)
		})
	}
}

var generateNewJobIDMockF = func() string {
	return "test-job-id"
}

func TestHandlePostCommand(t *testing.T) {
	generateNewJobID = generateNewJobIDMockF
	testJID := generateNewJobIDMockF()
	testUser := "test-user"

	defaultTimeout := 60 * time.Second
	gotCmd := "/bin/date;foo;whoami"
	gotCmdTimeoutSec := 30
	gotCmdTimeout := time.Duration(gotCmdTimeoutSec) * time.Second
	validReqBody := `{"command": "` + gotCmd + `","timeout_sec": ` + strconv.Itoa(gotCmdTimeoutSec) + `}`

	connMock := test.NewConnMock()
	// by default set to return success
	connMock.ReturnOk = true
	sshSuccessResp := comm.RunCmdResponse{Pid: 123, StartedAt: time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC)}
	sshRespBytes, err := json.Marshal(sshSuccessResp)
	require.NoError(t, err)
	connMock.ReturnResponsePayload = sshRespBytes

	s1 := sb.New(t).Connection(connMock).Build()
	s2 := sb.New(t).DisconnectedDuration(5 * time.Minute).Build()

	testCases := []struct {
		name string

		sid             string
		requestBody     string
		noJobProvider   bool
		jpReturnSaveErr error
		connReturnErr   error
		connReturnNotOk bool
		connReturnResp  []byte
		runningJob      *models.Job
		sessions        []*sessions.ClientSession

		wantStatusCode int
		wantTimeout    time.Duration
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
	}{
		{
			name:           "valid cmd",
			requestBody:    validReqBody,
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantStatusCode: http.StatusOK,
			wantTimeout:    gotCmdTimeout,
		},
		{
			name:           "valid cmd with no timeout",
			requestBody:    `{"command": "/bin/date;foo;whoami"}`,
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantTimeout:    defaultTimeout,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "valid cmd with 0 timeout",
			requestBody:    `{"command": "/bin/date;foo;whoami", "timeout_sec": 0}`,
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantTimeout:    defaultTimeout,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "empty cmd",
			requestBody:    `{"command": "", "timeout_sec": 30}`,
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Command cannot be empty.",
		},
		{
			name:           "no cmd",
			requestBody:    `{"timeout_sec": 30}`,
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Command cannot be empty.",
		},
		{
			name:           "empty body",
			requestBody:    "",
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Missing body with json data.",
		},
		{
			name:           "invalid request body",
			requestBody:    "sdfn fasld fasdf sdlf jd",
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Invalid JSON data.",
			wantErrDetail:  "invalid character 's' looking for beginning of value",
		},
		{
			name:           "no active session",
			requestBody:    validReqBody,
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{},
			wantStatusCode: http.StatusNotFound,
			wantErrTitle:   fmt.Sprintf("Active session with id=%q not found.", s1.ID),
		},
		{
			name:           "disconnected session",
			requestBody:    validReqBody,
			sid:            s2.ID,
			sessions:       []*sessions.ClientSession{s1, s2},
			wantStatusCode: http.StatusNotFound,
			wantErrTitle:   fmt.Sprintf("Active session with id=%q not found.", s2.ID),
		},
		{
			name:           "no persistent storage",
			requestBody:    validReqBody,
			noJobProvider:  true,
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantErrCode:    ErrCodeRunCmdDisabled,
			wantErrTitle:   "Persistent storage required. A data dir or a database table is required to activate this feature.",
		},
		{
			name:            "error on save job",
			requestBody:     validReqBody,
			jpReturnSaveErr: errors.New("save fake error"),
			sid:             s1.ID,
			sessions:        []*sessions.ClientSession{s1},
			wantStatusCode:  http.StatusInternalServerError,
			wantErrTitle:    "Failed to persist a new job.",
			wantErrDetail:   "save fake error",
		},
		{
			name:           "error on send request",
			requestBody:    validReqBody,
			connReturnErr:  errors.New("send fake error"),
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantStatusCode: http.StatusInternalServerError,
			wantErrTitle:   "Failed to execute remote command.",
			wantErrDetail:  "failed to send request: send fake error",
		},
		{
			name:           "invalid ssh response format",
			requestBody:    validReqBody,
			connReturnResp: []byte("invalid ssh response data"),
			sid:            s1.ID,
			sessions:       []*sessions.ClientSession{s1},
			wantStatusCode: http.StatusConflict,
			wantErrTitle:   "invalid client response format: failed to decode response into *comm.RunCmdResponse: invalid character 'i' looking for beginning of value",
		},
		{
			name:            "failure response on send request",
			requestBody:     validReqBody,
			connReturnNotOk: true,
			connReturnResp:  []byte("fake failure msg"),
			sid:             s1.ID,
			sessions:        []*sessions.ClientSession{s1},
			wantStatusCode:  http.StatusConflict,
			wantErrTitle:    "client error: fake failure msg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			al := APIListener{
				Server: &Server{
					sessionService: NewSessionService(nil, sessions.NewSessionRepository(tc.sessions, &hour)),
					config: &Config{
						Server: ServerConfig{
							RunRemoteCmdTimeout: defaultTimeout,
							MaxRequestBytes:     1024 * 1024,
						},
					},
				},
				Logger: testLog,
			}
			al.initRouter()

			jp := NewJobProviderMock()
			jp.ReturnErr = tc.jpReturnSaveErr
			if !tc.noJobProvider {
				al.jobProvider = jp
			}

			connMock.ReturnErr = tc.connReturnErr
			connMock.ReturnOk = !tc.connReturnNotOk
			if len(tc.connReturnResp) > 0 {
				connMock.ReturnResponsePayload = tc.connReturnResp // override stubbed success payload
			}

			ctx := api.WithUser(context.Background(), testUser)
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/sessions/%s/commands", tc.sid), strings.NewReader(tc.requestBody))
			req = req.WithContext(ctx)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantErrTitle == "" {
				// success case
				assert.Equal(t, fmt.Sprintf("{\"data\":{\"jid\":\"%s\"}}", testJID), w.Body.String())
				gotRunningJob := jp.InputJob
				assert.NotNil(t, gotRunningJob)
				assert.Equal(t, testJID, gotRunningJob.JID)
				assert.Equal(t, models.JobStatusRunning, gotRunningJob.Status)
				assert.Nil(t, gotRunningJob.FinishedAt)
				assert.Equal(t, tc.sid, gotRunningJob.SID)
				assert.Equal(t, gotCmd, gotRunningJob.Command)
				assert.Equal(t, sshSuccessResp.Pid, gotRunningJob.PID)
				assert.Equal(t, sshSuccessResp.StartedAt, gotRunningJob.StartedAt)
				assert.Equal(t, testUser, gotRunningJob.CreatedBy)
				assert.Equal(t, tc.wantTimeout, gotRunningJob.Timeout)
				assert.Nil(t, gotRunningJob.Result)
			} else {
				// failure case
				wantResp := api.NewErrorPayloadWithCode(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandleGetCommand(t *testing.T) {
	wantJob := jb.New(t).SID("sid-1234").JID("jid-1234").Build()
	wantJobResp := api.NewSuccessPayload(wantJob)
	b, err := json.Marshal(wantJobResp)
	require.NoError(t, err)
	wantJobRespJSON := string(b)

	testCases := []struct {
		name string

		noJobProvider bool
		jpReturnErr   error
		jpReturnJob   *models.Job

		wantStatusCode int
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
	}{
		{
			name:           "job found",
			jpReturnJob:    wantJob,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "not found",
			jpReturnJob:    nil,
			wantStatusCode: http.StatusNotFound,
			wantErrTitle:   fmt.Sprintf("Job[id=%q] not found.", wantJob.JID),
		},
		{
			name:           "error on get job",
			jpReturnErr:    errors.New("get job fake error"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrTitle:   fmt.Sprintf("Failed to find a job[id=%q].", wantJob.JID),
			wantErrDetail:  "get job fake error",
		},
		{
			name:           "no persistent storage",
			noJobProvider:  true,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantErrCode:    ErrCodeRunCmdDisabled,
			wantErrTitle:   "Persistent storage required. A data dir or a database table is required to activate this feature.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			al := APIListener{
				Logger: testLog,
				Server: &Server{
					config: &Config{
						Server: ServerConfig{MaxRequestBytes: 1024 * 1024},
					},
				},
			}
			al.initRouter()

			jp := NewJobProviderMock()
			jp.ReturnErr = tc.jpReturnErr
			jp.ReturnJob = tc.jpReturnJob
			if !tc.noJobProvider {
				al.jobProvider = jp
			}

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/sessions/%s/commands/%s", wantJob.SID, wantJob.JID), nil)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantErrTitle == "" {
				// success case
				assert.Equal(t, wantJobRespJSON, w.Body.String())
				assert.Equal(t, wantJob.SID, jp.InputSID)
				assert.Equal(t, wantJob.JID, jp.InputJID)
			} else {
				// failure case
				wantResp := api.NewErrorPayloadWithCode(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandleGetCommands(t *testing.T) {
	ft := time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC)
	testSID := "sid-1234"
	jb := jb.New(t).SID(testSID)
	job1 := jb.Status(models.JobStatusFinished).FinishedAt(ft).Build().JobSummary
	job2 := jb.Status(models.JobStatusUnknown).FinishedAt(ft.Add(-time.Hour)).Build().JobSummary
	job3 := jb.Status(models.JobStatusFailed).FinishedAt(ft.Add(time.Minute)).Build().JobSummary
	job4 := jb.Status(models.JobStatusRunning).Build().JobSummary
	jpSuccessReturnJobSummaries := []*models.JobSummary{&job1, &job2, &job3, &job4}
	wantSuccessResp := api.NewSuccessPayload([]*models.JobSummary{&job4, &job3, &job1, &job2}) // sorted in desc
	b, err := json.Marshal(wantSuccessResp)
	require.NoError(t, err)
	wantSuccessRespJobsJSON := string(b)

	testCases := []struct {
		name string

		noJobProvider        bool
		jpReturnErr          error
		jpReturnJobSummaries []*models.JobSummary

		wantStatusCode  int
		wantSuccessResp string
		wantErrCode     string
		wantErrTitle    string
		wantErrDetail   string
	}{
		{
			name:                 "found few jobs",
			jpReturnJobSummaries: jpSuccessReturnJobSummaries,
			wantSuccessResp:      wantSuccessRespJobsJSON,
			wantStatusCode:       http.StatusOK,
		},
		{
			name:                 "not found",
			jpReturnJobSummaries: []*models.JobSummary{},
			wantSuccessResp:      `{"data":[]}`,
			wantStatusCode:       http.StatusOK,
		},
		{
			name:           "error on get job summaries",
			jpReturnErr:    errors.New("get job summaries fake error"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrTitle:   fmt.Sprintf("Failed to get client jobs: session_id=%q.", testSID),
			wantErrDetail:  "get job summaries fake error",
		},
		{
			name:           "no persistent storage",
			noJobProvider:  true,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantErrCode:    ErrCodeRunCmdDisabled,
			wantErrTitle:   "Persistent storage required. A data dir or a database table is required to activate this feature.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			al := APIListener{
				Logger: testLog,
				Server: &Server{
					config: &Config{
						Server: ServerConfig{MaxRequestBytes: 1024 * 1024},
					},
				},
			}
			al.initRouter()

			jp := NewJobProviderMock()
			jp.ReturnErr = tc.jpReturnErr
			jp.ReturnJobSummaries = tc.jpReturnJobSummaries
			if !tc.noJobProvider {
				al.jobProvider = jp
			}

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/sessions/%s/commands", testSID), nil)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantErrTitle == "" {
				// success case
				assert.Equal(t, tc.wantSuccessResp, w.Body.String())
				assert.Equal(t, testSID, jp.InputSID)
			} else {
				// failure case
				wantResp := api.NewErrorPayloadWithCode(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandleGetSessions(t *testing.T) {
	s1 := sb.New(t).ID("session-1").ClientID(&cl1.ID).Build()
	s2 := sb.New(t).ID("session-2").ClientID(&cl1.ID).DisconnectedDuration(5 * time.Minute).Build()
	al := APIListener{
		Server: &Server{
			sessionService: NewSessionService(nil, sessions.NewSessionRepository([]*sessions.ClientSession{s1, s2}, &hour)),
			config: &Config{
				Server: ServerConfig{MaxRequestBytes: 1024 * 1024},
			},
		},
	}
	al.initRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/sessions", nil)
	al.router.ServeHTTP(w, req)

	expectedJSON := `{
   "data":[
      {
         "id":"session-1",
         "name":"Random Rport Client",
         "os":"Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
         "os_arch":"amd64",
         "os_family":"alpine",
         "os_kernel":"linux",
         "hostname":"alpine-3-10-tk-01",
         "ipv4":[
            "192.168.122.111"
         ],
         "ipv6":[
            "fe80::b84f:aff:fe59:a0b1"
         ],
         "tags":[
            "Linux",
            "Datacenter 1"
         ],
         "version":"0.1.12",
         "address":"88.198.189.161:50078",
         "tunnels":[
            {
               "lhost":"0.0.0.0",
               "lport":"2222",
               "rhost":"0.0.0.0",
               "rport":"22",
               "lport_random":false,
               "scheme":null,
               "acl":null,
               "id":"1"
            },
            {
               "lhost":"0.0.0.0",
               "lport":"4000",
               "rhost":"0.0.0.0",
               "rport":"80",
               "lport_random":false,
               "scheme":null,
               "acl":null,
               "id":"2"
            }
         ],
         "client":"user1"
      },
      {
         "id":"session-2",
         "name":"Random Rport Client",
         "os":"Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
         "os_arch":"amd64",
         "os_family":"alpine",
         "os_kernel":"linux",
         "hostname":"alpine-3-10-tk-01",
         "ipv4":[
            "192.168.122.111"
         ],
         "ipv6":[
            "fe80::b84f:aff:fe59:a0b1"
         ],
         "tags":[
            "Linux",
            "Datacenter 1"
         ],
         "version":"0.1.12",
         "address":"88.198.189.161:50078",
         "tunnels":[
            {
               "lhost":"0.0.0.0",
               "lport":"2222",
               "rhost":"0.0.0.0",
               "rport":"22",
               "lport_random":false,
               "scheme":null,
               "acl":null,
               "id":"1"
            },
            {
               "lhost":"0.0.0.0",
               "lport":"4000",
               "rhost":"0.0.0.0",
               "rport":"80",
               "lport_random":false,
               "scheme":null,
               "acl":null,
               "id":"2"
            }
         ],
         "disconnected":"2020-08-19T13:04:23+03:00",
         "client":"user1"
      }
   ]
}`

	assert.Equal(t, 200, w.Code)
	assert.JSONEq(t, expectedJSON, w.Body.String())
}
