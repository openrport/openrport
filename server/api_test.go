package chserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

var testLog = chshare.NewLogger("api-listener-test", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)

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

	req, err := http.NewRequest(http.MethodGet, "/clients", nil)
	require.NoError(err)

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
			clientCache:    tc.clientCache,
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
			Logger:          testLog,
			clientProvider:  tc.clientProvider,
			clientCache:     tc.clientCache,
			clientAuthWrite: tc.clientAuthWrite,
		}

		req, err := http.NewRequest(http.MethodPost, "/clients", tc.requestBody)
		require.NoErrorf(err, msg)

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

	s1 := &sessions.ClientSession{
		ID:       "aa1210c7-1899-491e-8e71-564cacaf1df8",
		Name:     "Random Rport Client 1",
		OS:       "Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
		Hostname: "alpine-3-10-tk-01",
		IPv4:     []string{"192.168.122.111"},
		IPv6:     []string{"fe80::b84f:aff:fe59:a0b1"},
		Tags:     []string{"Linux", "Datacenter 1"},
		Version:  "0.1.12",
		Address:  "88.198.189.161:50078",
		Tunnels: []*sessions.Tunnel{
			{
				ID: "1",
				Remote: chshare.Remote{
					LocalHost:  "0.0.0.0",
					LocalPort:  "2222",
					RemoteHost: "0.0.0.0",
					RemotePort: "22",
				},
			},
			{
				ID: "2",
				Remote: chshare.Remote{
					LocalHost:  "0.0.0.0",
					LocalPort:  "4000",
					RemoteHost: "0.0.0.0",
					RemotePort: "80",
				},
			},
		},
		Disconnected: nil,
		ClientID:     &cl1.ID,
		Connection:   mockConn,
	}

	s2DisconnectedTime, _ := time.Parse(time.RFC3339, "2020-08-19T13:04:23+03:00")
	sessions.Now = func() time.Time {
		return s2DisconnectedTime.Add(5 * time.Minute)
	}
	hour := time.Hour

	s2 := &sessions.ClientSession{
		ID:           "2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
		Name:         "Random Rport Client 2",
		OS:           "Linux alpine-3-10-tk-02 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
		Hostname:     "alpine-3-10-tk-02",
		IPv4:         []string{"192.168.122.112"},
		IPv6:         []string{"fe80::b84f:aff:fe59:a0b2"},
		Tags:         []string{"Linux", "Datacenter 2"},
		Version:      "0.1.12",
		Address:      "88.198.189.162:50078",
		Tunnels:      make([]*sessions.Tunnel, 0),
		Disconnected: &s2DisconnectedTime,
		ClientID:     &cl1.ID,
	}

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
			wantErrTitle:    fmt.Sprintf("Client expected to have no active session(s), got %d.", 1),
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
			wantErrTitle:    fmt.Sprintf("Client expected to have no disconnected session(s), got %d.", 1),
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
				Logger:          testLog,
				clientProvider:  tc.clientProvider,
				clientCache:     tc.clientCache,
				clientAuthWrite: tc.clientAuthWrite,
				sessionService:  NewSessionService(nil, sessions.NewSessionRepository(tc.sessions, &hour)),
			}
			mockConn.closed = false

			url := fmt.Sprintf("/clients/%s", tc.clientID)
			url += tc.urlSuffix
			req, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(err)

			// when
			router := mux.NewRouter()
			router.HandleFunc("/clients/{client_id}", al.handleDeleteClient)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

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
