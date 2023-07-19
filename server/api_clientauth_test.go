package chserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/server/clients"
	"github.com/realvnc-labs/rport/server/clients/clientdata"
	"github.com/realvnc-labs/rport/server/clientsauth"
	"github.com/realvnc-labs/rport/share/query"
)

var (
	cl1 = &clientsauth.ClientAuth{ID: "user1", Password: "pswd1"}
	cl2 = &clientsauth.ClientAuth{ID: "user2", Password: "pswd2"}
	cl3 = &clientsauth.ClientAuth{ID: "user3", Password: "pswd3"}
)

func TestHandleGetClientsAuth(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		descr string // Test Case Description

		provider clientsauth.Provider

		wantStatusCode  int
		wantClientsAuth []*clientsauth.ClientAuth
		wantErrCode     string
		wantErrTitle    string
		wantCount       int
		idFilter        string
	}{
		{
			descr:           "auth file, 3 clients",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			wantStatusCode:  http.StatusOK,
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3},
			wantCount:       3,
			idFilter:        "*",
		},
		{
			descr:           "auth file, 1 client, empty id",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			wantStatusCode:  http.StatusOK,
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3},
			wantCount:       0,
			idFilter:        "",
		},
		{
			descr:           "auth file, 3 clients, no results",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			wantStatusCode:  http.StatusOK,
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3},
			wantCount:       0,
			idFilter:        "Na0ahquaphe9",
		},
		{
			descr:           "auth, single client",
			provider:        clientsauth.NewSingleProvider(cl1.ID, cl1.Password),
			wantStatusCode:  http.StatusOK,
			wantClientsAuth: []*clientsauth.ClientAuth{cl1},
			wantCount:       1,
			idFilter:        "*",
		},
		{
			descr:           "auth, single client. no results",
			provider:        clientsauth.NewSingleProvider(cl1.ID, cl1.Password),
			wantStatusCode:  http.StatusOK,
			wantClientsAuth: []*clientsauth.ClientAuth{},
			wantCount:       0,
			idFilter:        "rie2IZ1aiPhe",
		},
		{
			descr:           "auth db, 3 clients",
			provider:        clientsauth.NewDatabaseMockProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			wantStatusCode:  http.StatusOK,
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3},
			wantCount:       3,
			idFilter:        "*",
		},
		{
			descr:           "auth db, 1 client, empty id",
			provider:        clientsauth.NewDatabaseMockProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			wantStatusCode:  http.StatusOK,
			wantClientsAuth: []*clientsauth.ClientAuth{cl1},
			wantCount:       0,
			idFilter:        "",
		},
		{
			descr:           "auth db, no reults",
			provider:        clientsauth.NewDatabaseMockProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			wantStatusCode:  http.StatusOK,
			wantClientsAuth: []*clientsauth.ClientAuth{},
			wantCount:       0,
			idFilter:        "sdfetzuj",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.descr, func(t *testing.T) {
			// given
			al := APIListener{
				Logger: testLog,
				Server: &Server{
					config: &chconfig.Config{
						API: chconfig.APIConfig{
							MaxRequestBytes: 1024 * 1024,
						},
					},
					clientAuthProvider: tc.provider,
				},
			}

			// when
			req := httptest.NewRequest(http.MethodGet, "/api/v1/clients-auth", nil)
			q := req.URL.Query()
			q.Add("page[limit]", "3")
			q.Add("filter[id]", tc.idFilter)
			req.URL.RawQuery = q.Encode()
			t.Logf("%s: URL tested: %s", tc.descr, req.URL.String())
			handler := http.HandlerFunc(al.handleGetClientsAuth)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			// then
			t.Logf("Got response: %s", w.Body)
			require.Equal(tc.wantStatusCode, w.Code)
			var wantResp interface{}
			if tc.wantErrTitle == "" {
				// success case
				if tc.wantCount > 0 {
					wantResp = &api.SuccessPayload{
						Data: tc.wantClientsAuth,
						Meta: api.NewMeta(tc.wantCount),
					}
				} else {
					wantResp = &api.SuccessPayload{
						Data: make([]int, 0),
						Meta: api.NewMeta(0),
					}
				}
			} else {
				// failure case
				wantResp = api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, "")
			}
			wantRespBytes, err := json.Marshal(wantResp)
			require.NoError(err)
			require.Equal(string(wantRespBytes), w.Body.String())
		})
	}
}

func TestHandlePostClientsAuth(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	composeRequestBody := func(id, pswd string) io.Reader {
		c := clientsauth.ClientAuth{ID: id, Password: pswd}
		b, err := json.Marshal(c)
		require.NoError(err)
		return bytes.NewBuffer(b)
	}
	cl4 := &clientsauth.ClientAuth{ID: "user4", Password: "pswd4"}

	testCases := []struct {
		descr string // Test Case Description

		provider        clientsauth.Provider
		clientAuthWrite bool
		requestBody     io.Reader

		wantStatusCode  int
		wantClientsAuth []*clientsauth.ClientAuth
		wantErrCode     string
		wantErrTitle    string
		wantErrDetail   string
	}{
		{
			descr:           "auth file, new valid client",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, cl4.Password),
			wantStatusCode:  http.StatusCreated,
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth db, new valid client",
			provider:        clientsauth.NewDatabaseMockProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, cl4.Password),
			wantStatusCode:  http.StatusCreated,
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, empty request body",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: true,
			requestBody:     strings.NewReader(""),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     "",
			wantErrTitle:    "Missing body with json data.",
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, invalid request body",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: true,
			requestBody:     strings.NewReader("invalid json"),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     "",
			wantErrTitle:    "Invalid JSON data.",
			wantErrDetail:   "invalid character 'i' looking for beginning of value",
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, invalid request, empty id",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: true,
			requestBody:     composeRequestBody("", cl4.Password),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing ID.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, invalid request, 'id' is missing",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: true,
			requestBody:     strings.NewReader(`{"password":"pswd"}`),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing ID.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, invalid request, empty password",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, ""),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing password.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, invalid request, 'password' is missing",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: true,
			requestBody:     strings.NewReader(`{"id":"user"}`),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing password.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, invalid request, id too short",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: true,
			requestBody:     composeRequestBody("12", cl4.Password),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing ID.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, invalid request, password too short",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, "12"),
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid or missing password.",
			wantErrDetail:   fmt.Sprintf("Min size is %d.", MinCredentialsLength),
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, client already exist",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl1.ID, cl4.Password),
			wantStatusCode:  http.StatusConflict,
			wantErrCode:     ErrCodeAlreadyExist,
			wantErrTitle:    fmt.Sprintf("Client Auth with ID %q already exist.", cl1.ID),
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth file, auth in Read-Only mode",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3, cl4}, t),
			clientAuthWrite: false,
			requestBody:     composeRequestBody(cl1.ID, cl4.Password),
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthRO,
			wantErrTitle:    "Client authentication has been attached in read-only mode.",
			wantClientsAuth: []*clientsauth.ClientAuth{cl1, cl2, cl3, cl4},
		},
		{
			descr:           "auth, single client",
			provider:        clientsauth.NewSingleProvider(cl1.ID, cl1.Password),
			clientAuthWrite: true,
			requestBody:     composeRequestBody(cl4.ID, cl4.Password),
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthSingleClient,
			wantErrTitle:    "Client authentication is enabled only for a single user.",
			wantClientsAuth: []*clientsauth.ClientAuth{cl1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.descr, func(t *testing.T) {
			// given
			al := APIListener{
				Server: &Server{
					config: &chconfig.Config{
						Server: chconfig.ServerConfig{
							AuthWrite: tc.clientAuthWrite,
						},
						API: chconfig.APIConfig{
							MaxRequestBytes: 1024 * 1024,
						},
					},
					clientAuthProvider: tc.provider,
				},
				Logger: testLog,
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/clients-auth", tc.requestBody)

			// when
			handler := http.HandlerFunc(al.handlePostClientsAuth)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			t.Logf("Got response %s", w.Body)

			// then
			require.Equal(tc.wantStatusCode, w.Code)
			if tc.wantErrTitle == "" {
				// success case
				assert.Empty(w.Body.String())
			} else {
				// failure case
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(err)
				require.Equal(string(wantRespBytes), w.Body.String())
			}
			filter := &query.ListOptions{
				Pagination: query.NewPagination(5, 0),
			}
			clients, _, err := al.clientAuthProvider.GetFiltered(filter)
			require.NoError(err)
			assert.ElementsMatch(tc.wantClientsAuth, clients)
		})
	}
}

func TestHandleDeleteClientAuth(t *testing.T) {
	mockConn := &mockConnection{}
	initState := []*clientsauth.ClientAuth{cl1, cl2, cl3}

	c1 := clients.New(t).ClientAuthID(cl1.ID).Connection(mockConn).Logger(testLog).Build()
	c2 := clients.New(t).ClientAuthID(cl1.ID).DisconnectedDuration(5 * time.Minute).Logger(testLog).Build()

	testCases := []struct {
		descr string // Test Case Description

		provider        clientsauth.Provider
		clients         []*clientdata.Client
		clientAuthWrite bool
		clientAuthID    string
		urlSuffix       string

		wantStatusCode  int
		wantClientsAuth []*clientsauth.ClientAuth
		wantErrCode     string
		wantErrTitle    string
		wantErrDetail   string
		wantClosedConn  bool
		wantClients     []*clientdata.Client
	}{
		{
			descr:           "auth file, success delete",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clientAuthWrite: true,
			clientAuthID:    cl1.ID,
			wantStatusCode:  http.StatusNoContent,
			wantClientsAuth: []*clientsauth.ClientAuth{cl2, cl3},
		},
		{
			descr:           "auth db, success delete",
			provider:        clientsauth.NewDatabaseMockProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clientAuthWrite: true,
			clientAuthID:    cl1.ID,
			wantStatusCode:  http.StatusNoContent,
			wantClientsAuth: []*clientsauth.ClientAuth{cl2, cl3},
		},
		{
			descr:           "auth file, missing client ID",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clientAuthWrite: true,
			clientAuthID:    "unknown-client-id",
			wantStatusCode:  http.StatusNotFound,
			wantErrCode:     ErrCodeClientAuthNotFound,
			wantErrTitle:    fmt.Sprintf("Client Auth with ID=%q not found.", "unknown-client-id"),
			wantClientsAuth: initState,
		},
		{
			descr:           "auth file, client has active client",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clients:         []*clientdata.Client{c1},
			clientAuthWrite: true,
			clientAuthID:    cl1.ID,
			wantStatusCode:  http.StatusConflict,
			wantErrCode:     ErrCodeClientAuthHasClient,
			wantErrTitle:    fmt.Sprintf("Client Auth expected to have no active or disconnected bound client(s), got %d.", 1),
			wantClientsAuth: initState,
			wantClients:     []*clientdata.Client{c1},
		},
		{
			descr:           "auth file, client auth has disconnected client",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clients:         []*clientdata.Client{c2},
			clientAuthWrite: true,
			clientAuthID:    cl1.ID,
			wantStatusCode:  http.StatusConflict,
			wantErrCode:     ErrCodeClientAuthHasClient,
			wantErrTitle:    fmt.Sprintf("Client Auth expected to have no active or disconnected bound client(s), got %d.", 1),
			wantClientsAuth: initState,
			wantClients:     []*clientdata.Client{c2},
		},
		{
			descr:           "auth file, auth in Read-Only mode",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clientAuthWrite: false,
			clientAuthID:    cl1.ID,
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthRO,
			wantErrTitle:    "Client authentication has been attached in read-only mode.",
			wantClientsAuth: initState,
		},
		{
			descr:           "auth file, client auth has active client, force",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clients:         []*clientdata.Client{c1},
			clientAuthWrite: true,
			clientAuthID:    cl1.ID,
			urlSuffix:       "?force=true",
			wantStatusCode:  http.StatusNoContent,
			wantClientsAuth: []*clientsauth.ClientAuth{cl2, cl3},
			wantClosedConn:  true,
		},
		{
			descr:           "auth file, client auth has disconnected bound client, force",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clients:         []*clientdata.Client{c2},
			clientAuthWrite: true,
			clientAuthID:    cl1.ID,
			urlSuffix:       "?force=true",
			wantStatusCode:  http.StatusNoContent,
			wantClientsAuth: []*clientsauth.ClientAuth{cl2, cl3},
		},
		{
			descr:           "invalid force param",
			provider:        clientsauth.NewMockFileProvider([]*clientsauth.ClientAuth{cl1, cl2, cl3}, t),
			clients:         []*clientdata.Client{c1, c2},
			clientAuthWrite: true,
			clientAuthID:    cl1.ID,
			urlSuffix:       "?force=test",
			wantStatusCode:  http.StatusBadRequest,
			wantErrCode:     ErrCodeInvalidRequest,
			wantErrTitle:    "Invalid force param test.",
			wantClientsAuth: initState,
			wantClients:     []*clientdata.Client{c1, c2},
		},
		{
			descr:           "auth, single client",
			provider:        clientsauth.NewSingleProvider(cl1.ID, cl1.Password),
			clientAuthWrite: true,
			clientAuthID:    cl1.ID,
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantErrCode:     ErrCodeClientAuthSingleClient,
			wantErrTitle:    "Client authentication is enabled only for a single user.",
			wantClientsAuth: []*clientsauth.ClientAuth{cl1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.descr, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			// given
			al := APIListener{
				insecureForTests: true,
				Server: &Server{
					clientService: clients.NewClientService(nil, nil, clients.NewClientRepository(tc.clients, &hour, testLog), testLog, nil),
					config: &chconfig.Config{
						Server: chconfig.ServerConfig{
							AuthWrite: tc.clientAuthWrite,
						},
						API: chconfig.APIConfig{
							MaxRequestBytes: 1024 * 1024,
						},
					},
					clientAuthProvider: tc.provider,
				},
				Logger: testLog,
			}
			al.initRouter()
			mockConn.closed = false

			url := fmt.Sprintf("/api/v1/clients-auth/%s", tc.clientAuthID)
			url += tc.urlSuffix
			req := httptest.NewRequest(http.MethodDelete, url, nil)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)
			t.Logf("got response %s", w.Body.String())

			// then
			assert.Equal(tc.wantStatusCode, w.Code)
			var wantRespStr string
			if tc.wantErrTitle == "" {
				// success case: empty body
			} else {
				// failure case
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(err)
				wantRespStr = string(wantRespBytes)
			}
			assert.Equal(wantRespStr, w.Body.String())
			filter := &query.ListOptions{
				Pagination: query.NewPagination(5, 0),
			}
			clients, _, err := al.clientAuthProvider.GetFiltered(filter)
			require.NoError(err)
			assert.ElementsMatch(tc.wantClientsAuth, clients, "clients auth not as expected")
			assert.Equal(tc.wantClosedConn, mockConn.closed)
			allClients := al.clientService.GetAll()
			assert.ElementsMatch(tc.wantClients, allClients)
		})
	}
}
