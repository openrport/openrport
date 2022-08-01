package chserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/db/migration/client_groups"
	jobsmigration "github.com/cloudradar-monitoring/rport/db/migration/jobs"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/api/jobs/schedule"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/test/jb"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
	"github.com/cloudradar-monitoring/rport/share/random"
	"github.com/cloudradar-monitoring/rport/share/security"
	"github.com/cloudradar-monitoring/rport/share/test"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

type JobProviderMock struct {
	JobProvider
	ReturnJob     *models.Job
	ReturnJobList []*models.Job
	ReturnErr     error

	InputCID       string
	InputJID       string
	InputSaveJob   *models.Job
	InputCreateJob *models.Job
}

func NewJobProviderMock() *JobProviderMock {
	return &JobProviderMock{}
}

func (p *JobProviderMock) GetByJID(cid, jid string) (*models.Job, error) {
	p.InputCID = cid
	p.InputJID = jid
	return p.ReturnJob, p.ReturnErr
}

func (p *JobProviderMock) List(ctx context.Context, opts *query.ListOptions) ([]*models.Job, error) {
	p.InputCID = opts.Filters[0].Values[0]
	return p.ReturnJobList, p.ReturnErr
}

func (p *JobProviderMock) Count(ctx context.Context, opts *query.ListOptions) (int, error) {
	return len(p.ReturnJobList), p.ReturnErr
}

func (p *JobProviderMock) SaveJob(job *models.Job) error {
	p.InputSaveJob = job
	return p.ReturnErr
}

func (p *JobProviderMock) CreateJob(job *models.Job) error {
	p.InputCreateJob = job
	return p.ReturnErr
}

func (p *JobProviderMock) Close() error {
	return nil
}

func TestHandlePostCommand(t *testing.T) {
	var testJID string
	generateNewJobID = func() (string, error) {
		uuid, err := random.UUID4()
		testJID = uuid
		return uuid, err
	}
	testUser := "test-user"

	defaultTimeout := 60
	gotCmd := "/bin/date;foo;whoami"
	gotCmdTimeoutSec := 30
	validReqBody := `{"command": "` + gotCmd + `","timeout_sec": ` + strconv.Itoa(gotCmdTimeoutSec) + `}`

	connMock := test.NewConnMock()
	// by default set to return success
	connMock.ReturnOk = true
	sshSuccessResp := comm.RunCmdResponse{Pid: 123, StartedAt: time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC)}
	sshRespBytes, err := json.Marshal(sshSuccessResp)
	require.NoError(t, err)
	connMock.ReturnResponsePayload = sshRespBytes

	c1 := clients.New(t).Connection(connMock).Build()
	c2 := clients.New(t).DisconnectedDuration(5 * time.Minute).Build()

	testCases := []struct {
		name string

		cid             string
		requestBody     string
		jpReturnSaveErr error
		connReturnErr   error
		connReturnNotOk bool
		connReturnResp  []byte
		runningJob      *models.Job
		clients         []*clients.Client

		wantStatusCode  int
		wantTimeout     int
		wantErrCode     string
		wantErrTitle    string
		wantErrDetail   string
		wantInterpreter string
	}{
		{
			name:           "valid cmd",
			requestBody:    validReqBody,
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantStatusCode: http.StatusOK,
			wantTimeout:    gotCmdTimeoutSec,
		},
		{
			name:            "valid cmd with interpreter",
			requestBody:     `{"command": "` + gotCmd + `","interpreter": "powershell"}`,
			cid:             c1.ID,
			clients:         []*clients.Client{c1},
			wantStatusCode:  http.StatusOK,
			wantTimeout:     defaultTimeout,
			wantInterpreter: "powershell",
		},
		{
			name:           "invalid interpreter",
			requestBody:    `{"command": "` + gotCmd + `","interpreter": "unsupported"}`,
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Invalid interpreter.",
			wantErrDetail:  "expected interpreter to be one of: [cmd powershell tacoscript], actual: unsupported",
		},
		{
			name:           "valid cmd with no timeout",
			requestBody:    `{"command": "/bin/date;foo;whoami"}`,
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantTimeout:    defaultTimeout,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "valid cmd with 0 timeout",
			requestBody:    `{"command": "/bin/date;foo;whoami", "timeout_sec": 0}`,
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantTimeout:    defaultTimeout,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "empty cmd",
			requestBody:    `{"command": "", "timeout_sec": 30}`,
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Command cannot be empty.",
		},
		{
			name:           "no cmd",
			requestBody:    `{"timeout_sec": 30}`,
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Command cannot be empty.",
		},
		{
			name:           "empty body",
			requestBody:    "",
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Missing body with json data.",
		},
		{
			name:           "invalid request body",
			requestBody:    "sdfn fasld fasdf sdlf jd",
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Invalid JSON data.",
			wantErrDetail:  "invalid character 's' looking for beginning of value",
		},
		{
			name:           "invalid request body: unknown param",
			requestBody:    `{"command": "/bin/date;foo;whoami", "timeout": 30}`,
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Invalid JSON data.",
			wantErrDetail:  "json: unknown field \"timeout\"",
		},
		{
			name:           "no active client",
			requestBody:    validReqBody,
			cid:            c1.ID,
			clients:        []*clients.Client{},
			wantStatusCode: http.StatusNotFound,
			wantErrTitle:   fmt.Sprintf("Active client with id=%q not found.", c1.ID),
		},
		{
			name:           "disconnected client",
			requestBody:    validReqBody,
			cid:            c2.ID,
			clients:        []*clients.Client{c1, c2},
			wantStatusCode: http.StatusNotFound,
			wantErrTitle:   fmt.Sprintf("Active client with id=%q not found.", c2.ID),
		},
		{
			name:            "error on save job",
			requestBody:     validReqBody,
			jpReturnSaveErr: errors.New("save fake error"),
			cid:             c1.ID,
			clients:         []*clients.Client{c1},
			wantStatusCode:  http.StatusInternalServerError,
			wantErrTitle:    "Failed to persist a new job.",
			wantErrDetail:   "save fake error",
		},
		{
			name:           "error on send request",
			requestBody:    validReqBody,
			connReturnErr:  errors.New("send fake error"),
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantStatusCode: http.StatusInternalServerError,
			wantErrTitle:   "Failed to execute remote command.",
			wantErrDetail:  "failed to send request: send fake error",
		},
		{
			name:           "invalid ssh response format",
			requestBody:    validReqBody,
			connReturnResp: []byte("invalid ssh response data"),
			cid:            c1.ID,
			clients:        []*clients.Client{c1},
			wantStatusCode: http.StatusConflict,
			wantErrTitle:   "invalid client response format: failed to decode response into *comm.RunCmdResponse: invalid character 'i' looking for beginning of value",
		},
		{
			name:            "failure response on send request",
			requestBody:     validReqBody,
			connReturnNotOk: true,
			connReturnResp:  []byte("fake failure msg"),
			cid:             c1.ID,
			clients:         []*clients.Client{c1},
			wantStatusCode:  http.StatusConflict,
			wantErrTitle:    "client error: fake failure msg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			al := APIListener{
				insecureForTests: true,
				Server: &Server{
					clientService: NewClientService(nil, nil, clients.NewClientRepository(tc.clients, &hour, testLog)),
					config: &Config{
						Server: ServerConfig{
							RunRemoteCmdTimeoutSec: defaultTimeout,
							MaxRequestBytes:        1024 * 1024,
						},
					},
				},
				Logger: testLog,
			}
			al.initRouter()

			jp := NewJobProviderMock()
			jp.ReturnErr = tc.jpReturnSaveErr
			al.jobProvider = jp

			connMock.ReturnErr = tc.connReturnErr
			connMock.ReturnOk = !tc.connReturnNotOk
			if len(tc.connReturnResp) > 0 {
				connMock.ReturnResponsePayload = tc.connReturnResp // override stubbed success payload
			}

			ctx := api.WithUser(context.Background(), testUser)
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clients/%s/commands", tc.cid), strings.NewReader(tc.requestBody))
			req = req.WithContext(ctx)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantErrTitle == "" {
				// success case
				assert.Equal(t, fmt.Sprintf("{\"data\":{\"jid\":\"%s\"}}", testJID), w.Body.String())
				gotRunningJob := jp.InputCreateJob
				assert.NotNil(t, gotRunningJob)
				assert.Equal(t, testJID, gotRunningJob.JID)
				assert.Equal(t, models.JobStatusRunning, gotRunningJob.Status)
				assert.Nil(t, gotRunningJob.FinishedAt)
				assert.Equal(t, tc.cid, gotRunningJob.ClientID)
				assert.Equal(t, gotCmd, gotRunningJob.Command)
				assert.Equal(t, tc.wantInterpreter, gotRunningJob.Interpreter)
				assert.Equal(t, &sshSuccessResp.Pid, gotRunningJob.PID)
				assert.Equal(t, sshSuccessResp.StartedAt, gotRunningJob.StartedAt)
				assert.Equal(t, testUser, gotRunningJob.CreatedBy)
				assert.Equal(t, tc.wantTimeout, gotRunningJob.TimeoutSec)
				assert.Nil(t, gotRunningJob.Result)
			} else {
				// failure case
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandleGetCommand(t *testing.T) {
	wantJob := jb.New(t).ClientID("cid-1234").JID("jid-1234").Build()
	wantJobResp := api.NewSuccessPayload(wantJob)
	b, err := json.Marshal(wantJobResp)
	require.NoError(t, err)
	wantJobRespJSON := string(b)

	testCases := []struct {
		name string

		jpReturnErr error
		jpReturnJob *models.Job

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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			al := APIListener{
				insecureForTests: true,
				Logger:           testLog,
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
			al.jobProvider = jp

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/clients/%s/commands/%s", wantJob.ClientID, wantJob.JID), nil)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantErrTitle == "" {
				// success case
				assert.Equal(t, wantJobRespJSON, w.Body.String())
				assert.Equal(t, wantJob.ClientID, jp.InputCID)
				assert.Equal(t, wantJob.JID, jp.InputJID)
			} else {
				// failure case
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandleGetCommands(t *testing.T) {
	ft := time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC)
	testCID := "cid-1234"
	jb := jb.New(t).ClientID(testCID)
	job1 := jb.Status(models.JobStatusSuccessful).FinishedAt(ft).Build()
	job2 := jb.Status(models.JobStatusUnknown).FinishedAt(ft.Add(-time.Hour)).Build()
	job3 := jb.Status(models.JobStatusFailed).FinishedAt(ft.Add(time.Minute)).Build()
	job4 := jb.Status(models.JobStatusRunning).Build()
	wantResp1 := fmt.Sprintf(
		`{"data":[{"jid":"%s"},{"jid":"%s"},{"jid":"%s"},{"jid":"%s"}], "meta": {"count": 4}}`,
		job1.JID,
		job2.JID,
		job3.JID,
		job4.JID,
	)
	wantResp2 := fmt.Sprintf(
		`{"data":[{"jid":"%s", "finished_at": "%s", "status": "%s", "result":{"summary":"%s"}}], "meta": {"count": 1}}`,
		job1.JID,
		job1.FinishedAt.Format(time.RFC3339),
		job1.Status,
		job1.Result.Summary,
	)

	testCases := []struct {
		name   string
		params string

		jpReturnErr  error
		jpReturnJobs []*models.Job

		wantStatusCode  int
		wantSuccessResp string
		wantErrCode     string
		wantErrTitle    string
		wantErrDetail   string
	}{
		{
			name:            "found few jobs, jid only",
			params:          "fields[commands]=jid",
			jpReturnJobs:    []*models.Job{job1, job2, job3, job4},
			wantSuccessResp: wantResp1,
			wantStatusCode:  http.StatusOK,
		},
		{
			name:            "found one job, default fields",
			jpReturnJobs:    []*models.Job{job1},
			wantSuccessResp: wantResp2,
			wantStatusCode:  http.StatusOK,
		},
		{
			name:            "not found",
			jpReturnJobs:    []*models.Job{},
			wantSuccessResp: `{"data":[], "meta": {"count": 0}}`,
			wantStatusCode:  http.StatusOK,
		},
		{
			name:           "error on get job list",
			jpReturnErr:    errors.New("get job list fake error"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrTitle:   fmt.Sprintf("Failed to get client jobs: client_id=%q.", testCID),
			wantErrDetail:  "get job list fake error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			al := APIListener{
				insecureForTests: true,
				Logger:           testLog,
				Server: &Server{
					config: &Config{
						Server: ServerConfig{MaxRequestBytes: 1024 * 1024},
					},
				},
			}
			al.initRouter()

			jp := NewJobProviderMock()
			jp.ReturnErr = tc.jpReturnErr
			jp.ReturnJobList = tc.jpReturnJobs
			al.jobProvider = jp

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/clients/%s/commands?%s", testCID, tc.params), nil)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantErrTitle == "" {
				// success case
				assert.JSONEq(t, tc.wantSuccessResp, w.Body.String())
				assert.Equal(t, testCID, jp.InputCID)
			} else {
				// failure case
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandlePostMultiClientCommand(t *testing.T) {
	testUser := "test-user"
	curUser := &users.User{
		Username: testUser,
		Groups:   []string{users.Administrators},
	}

	connMock1 := test.NewConnMock()
	// by default set to return success
	connMock1.ReturnOk = true
	sshSuccessResp1 := comm.RunCmdResponse{Pid: 1, StartedAt: time.Date(2020, 10, 10, 10, 10, 1, 0, time.UTC)}
	sshRespBytes1, err := json.Marshal(sshSuccessResp1)
	require.NoError(t, err)
	connMock1.ReturnResponsePayload = sshRespBytes1

	connMock2 := test.NewConnMock()
	// by default set to return success
	connMock2.ReturnOk = true
	sshSuccessResp2 := comm.RunCmdResponse{Pid: 2, StartedAt: time.Date(2020, 10, 10, 10, 10, 2, 0, time.UTC)}
	sshRespBytes2, err := json.Marshal(sshSuccessResp2)
	require.NoError(t, err)
	connMock2.ReturnResponsePayload = sshRespBytes2

	c1 := clients.New(t).ID("client-1").Connection(connMock1).Build()
	c2 := clients.New(t).ID("client-2").Connection(connMock2).Build()
	c3 := clients.New(t).ID("client-3").DisconnectedDuration(5 * time.Minute).Build()

	defaultTimeout := 60
	gotCmd := "/bin/date;foo;whoami"
	gotCmdTimeoutSec := 30
	validReqBody := `{"command": "` + gotCmd +
		`","timeout_sec": ` + strconv.Itoa(gotCmdTimeoutSec) +
		`,"client_ids": ["` + c1.ID + `", "` + c2.ID + `"]` +
		`,"abort_on_error": false` +
		`,"execute_concurrently": false` +
		`}`

	testCases := []struct {
		name string

		requestBody string
		abortOnErr  bool

		connReturnErr error

		wantStatusCode int
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
		wantJobErr     string
	}{
		{
			name:           "valid cmd",
			requestBody:    validReqBody,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "no targeting params provided",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Missing targeting parameters.",
			wantErrDetail:  ErrRequestMissingTargetingParams.Error(),
		},
		{
			name: "only one client",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"client_ids": ["client-1"]
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "At least 2 clients should be specified.",
		},
		{
			name: "disconnected client",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"client_ids": ["client-1", "client-3"]
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   fmt.Sprintf("Client with id=%q is not active.", c3.ID),
		},
		{
			name: "client not found",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"client_ids": ["client-1", "client-4"]
		}`,
			wantStatusCode: http.StatusNotFound,
			wantErrTitle:   fmt.Sprintf("Client with id=%q not found.", "client-4"),
		},
		{
			name:           "error on send request",
			requestBody:    validReqBody,
			connReturnErr:  errors.New("send fake error"),
			wantStatusCode: http.StatusOK,
			wantJobErr:     "failed to send request: send fake error",
		},
		{
			name: "error on send request, abort on err",
			requestBody: `
			{
				"command": "/bin/date;foo;whoami",
				"timeout_sec": 30,
				"client_ids": ["client-1", "client-2"],
				"execute_concurrently": false,
				"abort_on_error": true
			}`,
			abortOnErr:     true,
			connReturnErr:  errors.New("send fake error"),
			wantStatusCode: http.StatusOK,
			wantJobErr:     "failed to send request: send fake error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			al := APIListener{
				insecureForTests: true,
				Server: &Server{
					clientService: NewClientService(nil, nil, clients.NewClientRepository([]*clients.Client{c1, c2, c3}, &hour, testLog)),
					config: &Config{
						Server: ServerConfig{
							RunRemoteCmdTimeoutSec: defaultTimeout,
							MaxRequestBytes:        1024 * 1024,
						},
					},
					jobsDoneChannel: jobResultChanMap{
						m: make(map[string]chan *models.Job),
					},
				},
				userService: users.NewAPIService(users.NewStaticProvider([]*users.User{curUser}), false),
				Logger:      testLog,
			}
			var done chan bool
			if tc.wantStatusCode == http.StatusOK {
				done = make(chan bool)
				al.testDone = done
			}

			al.initRouter()

			jobsDB, err := sqlite.New(
				":memory:",
				jobsmigration.AssetNames(),
				jobsmigration.Asset,
				DataSourceOptions,
			)
			require.NoError(t, err)
			jp := jobs.NewSqliteProvider(jobsDB, testLog)
			defer jp.Close()
			al.jobProvider = jp

			connMock1.ReturnErr = tc.connReturnErr

			ctx := api.WithUser(context.Background(), testUser)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/commands", strings.NewReader(tc.requestBody))
			req = req.WithContext(ctx)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantStatusCode == http.StatusOK {
				// wait until async task executeMultiClientJob finishes
				<-al.testDone
			}
			if tc.wantErrTitle == "" {
				// success case
				assert.Contains(t, w.Body.String(), `{"data":{"jid":`)
				gotResp := api.NewSuccessPayload(newJobResponse{})
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotResp))
				gotPropMap, ok := gotResp.Data.(map[string]interface{})
				require.True(t, ok)
				jidObj, found := gotPropMap["jid"]
				require.True(t, found)
				gotJID, ok := jidObj.(string)
				require.True(t, ok)
				require.NotEmpty(t, gotJID)

				gotMultiJob, err := jp.GetMultiJob(ctx, gotJID)
				require.NoError(t, err)
				require.NotNil(t, gotMultiJob)
				if tc.abortOnErr {
					require.Len(t, gotMultiJob.Jobs, 1)
				} else {
					require.Len(t, gotMultiJob.Jobs, 2)
				}
				if tc.connReturnErr != nil {
					assert.Equal(t, models.JobStatusFailed, gotMultiJob.Jobs[0].Status)
					assert.Equal(t, tc.wantJobErr, gotMultiJob.Jobs[0].Error)
				} else {
					assert.Equal(t, models.JobStatusRunning, gotMultiJob.Jobs[0].Status)
				}
				if !tc.abortOnErr {
					assert.Equal(t, models.JobStatusRunning, gotMultiJob.Jobs[1].Status)
				}
			} else {
				// failure case
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandlePostMultiClientCommandWithGroupIDs(t *testing.T) {
	testUser := "test-user"
	defaultTimeout := 60

	testCases := []struct {
		name string

		requestBody string

		wantStatusCode int
		wantJobCount   int
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
	}{
		{
			name: "valid when group id with at least 2 clients",
			requestBody: `{
				"command": "/bin/date;foo;whoami",
				"timeout_sec": 30,
				"group_ids": ["group-1"],
				"abort_on_error": false,
				"execute_concurrently": false
			}`,
			wantStatusCode: http.StatusOK,
			wantJobCount:   2,
		},
		{
			name: "invalid when empty group ids",
			requestBody: `{
				"command": "/bin/date;foo;whoami",
				"timeout_sec": 30,
				"group_ids": [],
				"abort_on_error": false,
				"execute_concurrently": false
			}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "At least 2 clients should be specified.",
		},
		{
			name: "valid when group id with 1 client",
			requestBody: `{
				"command": "/bin/date;foo;whoami",
				"timeout_sec": 30,
				"group_ids": ["group-2"],
				"abort_on_error": false,
				"execute_concurrently": false
			}`,
			wantStatusCode: http.StatusOK,
			wantJobCount:   1,
		},
		{
			name: "valid when group id and client id",
			requestBody: `{
				"command": "/bin/date;foo;whoami",
				"timeout_sec": 30,
				"client_ids": ["client-1"],
				"group_ids": ["group-2"],
				"abort_on_error": false,
				"execute_concurrently": false
			}`,
			wantStatusCode: http.StatusOK,
			wantJobCount:   2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			curUser := makeTestUser(testUser)

			connMock1 := makeConnMock(t, true, 1, time.Date(2020, 10, 10, 10, 10, 1, 0, time.UTC))
			connMock2 := makeConnMock(t, true, 2, time.Date(2020, 10, 10, 10, 10, 2, 0, time.UTC))
			connMock4 := makeConnMock(t, true, 4, time.Date(2020, 10, 10, 10, 10, 4, 0, time.UTC))

			c1 := clients.New(t).ID("client-1").Connection(connMock1).Build()
			c2 := clients.New(t).ID("client-2").Connection(connMock2).Build()
			c3 := clients.New(t).ID("client-3").DisconnectedDuration(5 * time.Minute).Build()
			c4 := clients.New(t).ID("client-4").Connection(connMock4).Build()

			g1 := makeClientGroup("group-1", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-1", "client-2"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			g2 := makeClientGroup("group-2", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-4"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			c1.AllowedUserGroups = []string{"group-1"}
			c2.AllowedUserGroups = []string{"group-1"}
			c4.AllowedUserGroups = []string{"group-2"}

			al := makeAPIListener(nil,
				curUser,
				clients.NewClientRepository([]*clients.Client{c1, c2, c3, c4}, &hour, testLog),
				defaultTimeout,
				testLog)

			var done chan bool
			if tc.wantStatusCode == http.StatusOK {
				done = make(chan bool)
				al.testDone = done
			}

			jp := makeJobsProvider(t, DataSourceOptions, testLog)
			defer jp.Close()

			gp := makeGroupsProvider(t, DataSourceOptions, testLog)
			defer gp.Close()

			al.initRouter()

			al.jobProvider = jp
			al.clientGroupProvider = gp

			ctx := api.WithUser(context.Background(), testUser)

			err := gp.Create(ctx, g1)
			assert.NoError(t, err)
			err = gp.Create(ctx, g2)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/commands", strings.NewReader(tc.requestBody))
			req = req.WithContext(ctx)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantStatusCode == http.StatusOK {
				// wait until async task executeMultiClientJob finishes
				<-al.testDone
			}
			if tc.wantErrTitle == "" {
				// success case
				assert.Contains(t, w.Body.String(), `{"data":{"jid":`)
				gotResp := api.NewSuccessPayload(newJobResponse{})
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotResp))
				gotPropMap, ok := gotResp.Data.(map[string]interface{})
				require.True(t, ok)
				jidObj, found := gotPropMap["jid"]
				require.True(t, found)
				gotJID, ok := jidObj.(string)
				require.True(t, ok)
				require.NotEmpty(t, gotJID)

				gotMultiJob, err := jp.GetMultiJob(ctx, gotJID)
				require.NoError(t, err)
				require.NotNil(t, gotMultiJob)
				require.Len(t, gotMultiJob.Jobs, tc.wantJobCount)
			} else {
				// failure case
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandlePostMultiClientCommandWithTags(t *testing.T) {
	testUser := "test-user"
	defaultTimeout := 60

	testCases := []struct {
		name string

		requestBody string

		wantStatusCode int
		wantJobCount   int
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
	}{
		{
			name: "valid when only tags included",
			requestBody: `{
				"command": "/bin/date;foo;whoami",
				"timeout_sec": 30,
				"tags": {
					"tags": [
						"linux"
					],
					"operator": "OR"
				},
				"abort_on_error": false,
				"execute_concurrently": false
			}`,
			wantStatusCode: http.StatusOK,
			wantJobCount:   2,
		},
		{
			name: "valid when only tags included and missing operator",
			requestBody: `{
				"command": "/bin/date;foo;whoami",
				"timeout_sec": 30,
				"tags": {
					"tags": [
						"linux"
					]
				},
				"abort_on_error": false,
				"execute_concurrently": false
			}`,
			wantStatusCode: http.StatusOK,
			wantJobCount:   2,
		},
		{
			name: "error when client ids and tags included",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"client_ids": ["client-1", "client-2"],
			"tags": {
				"tags": [
					"linux", 
					"windows"
				],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Multiple targeting parameters.",
			wantErrDetail:  ErrRequestIncludesMultipleTargetingParams.Error(),
		},
		{
			name: "error when empty tags",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"tags": {
				"tags": [],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "No tags specified.",
			wantErrDetail:  ErrMissingTagsInMultiJobRequest.Error(),
		},
		{
			name: "error when no clients for tag",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"tags": {
				"tags": ["random"],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "At least 1 client should be specified.",
		},
		{
			name: "error when group ids and tags included",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"group_ids": ["group-1"],
			"tags": {
				"tags": [
					"linux", 
					"windows"
				],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Multiple targeting parameters.",
			wantErrDetail:  ErrRequestIncludesMultipleTargetingParams.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			curUser := makeTestUser(testUser)

			connMock1 := makeConnMock(t, true, 1, time.Date(2020, 10, 10, 10, 10, 1, 0, time.UTC))
			connMock2 := makeConnMock(t, true, 2, time.Date(2020, 10, 10, 10, 10, 2, 0, time.UTC))
			connMock4 := makeConnMock(t, true, 4, time.Date(2020, 10, 10, 10, 10, 4, 0, time.UTC))

			c1 := clients.New(t).ID("client-1").Connection(connMock1).Build()
			c2 := clients.New(t).ID("client-2").Connection(connMock2).Build()
			c3 := clients.New(t).ID("client-3").DisconnectedDuration(5 * time.Minute).Build()
			c4 := clients.New(t).ID("client-4").Connection(connMock4).Build()

			c1.Tags = []string{"linux"}
			c2.Tags = []string{"windows"}
			c3.Tags = []string{"mac"}
			c4.Tags = []string{"linux", "windows"}

			g1 := makeClientGroup("group-1", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-1", "client-2"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			g2 := makeClientGroup("group-2", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-4"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			c1.AllowedUserGroups = []string{"group-1"}
			c2.AllowedUserGroups = []string{"group-1"}
			c4.AllowedUserGroups = []string{"group-2"}

			clientList := []*clients.Client{c1, c2, c4}

			p := clients.NewFakeClientProvider(t, nil, nil)

			al := makeAPIListener(p,
				curUser,
				clients.NewClientRepositoryWithDB(nil, &hour, p, testLog),
				defaultTimeout,
				testLog)

			// make sure the repo has the test clients
			for _, cl := range clientList {
				err := al.clientService.GetRepo().Save(cl)
				assert.NoError(t, err)
			}

			var done chan bool
			if tc.wantStatusCode == http.StatusOK {
				done = make(chan bool)
				al.testDone = done
			}

			jp := makeJobsProvider(t, DataSourceOptions, testLog)
			defer jp.Close()

			gp := makeGroupsProvider(t, DataSourceOptions, testLog)
			defer gp.Close()

			al.initRouter()

			al.jobProvider = jp
			al.clientGroupProvider = gp

			ctx := api.WithUser(context.Background(), testUser)

			err := gp.Create(ctx, g1)
			assert.NoError(t, err)
			err = gp.Create(ctx, g2)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/commands", strings.NewReader(tc.requestBody))
			req = req.WithContext(ctx)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantStatusCode == http.StatusOK {
				// wait until async task executeMultiClientJob finishes
				<-al.testDone
			}
			if tc.wantErrTitle == "" {
				// success case
				assert.Contains(t, w.Body.String(), `{"data":{"jid":`)
				gotResp := api.NewSuccessPayload(newJobResponse{})
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotResp))
				gotPropMap, ok := gotResp.Data.(map[string]interface{})
				require.True(t, ok)
				jidObj, found := gotPropMap["jid"]
				require.True(t, found)
				gotJID, ok := jidObj.(string)
				require.True(t, ok)
				require.NotEmpty(t, gotJID)

				gotMultiJob, err := jp.GetMultiJob(ctx, gotJID)
				require.NoError(t, err)
				require.NotNil(t, gotMultiJob)
				require.Len(t, gotMultiJob.Jobs, tc.wantJobCount)
			} else {
				// failure case
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandlePostMultiClientWSCommandWithTags(t *testing.T) {
	testUser := "test-user"
	testToken := "12345678"
	defaultTimeout := 60

	testCases := []struct {
		name string

		requestBody string

		shouldSucceed bool
		wantJobCount  int
		wantErrCode   string
		wantErrTitle  string
		wantErrDetail string
	}{
		{
			name: "valid with client ids",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"client_ids": ["client-1", "client-2"],
			"abort_on_error": false,
			"execute_concurrently": false
		}`,
			shouldSucceed: true,
			wantJobCount:  2,
		},
		{
			name: "no targeting params provided",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30
		}`,
			shouldSucceed: false,
			wantErrTitle:  "Missing targeting parameters.",
			wantErrDetail: ErrRequestMissingTargetingParams.Error(),
		},
		{
			name: "valid when only tags included",
			requestBody: `{
				"command": "/bin/date;foo;whoami",
				"timeout_sec": 30,
				"tags": {
					"tags": [
						"linux"
					],
					"operator": "OR"
				},
				"abort_on_error": false,
				"execute_concurrently": false
			}`,
			shouldSucceed: true,
			wantJobCount:  2,
		},
		{
			name: "error when empty tags",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"tags": {
				"tags": [],
				"operator": "OR"
			}
		}`,
			shouldSucceed: false,
			wantErrTitle:  "No tags specified.",
			wantErrDetail: ErrMissingTagsInMultiJobRequest.Error(),
		},
		{
			name: "error when no clients for tag",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"tags": {
				"tags": ["random"],
				"operator": "OR"
			}
		}`,
			shouldSucceed: false,
			wantErrTitle:  "At least 1 client should be specified.",
		},
		{
			name: "error when client ids and tags included",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"client_ids": ["client-1", "client-2"],
			"tags": {
				"tags": [
					"linux",
					"windows"
				],
				"operator": "OR"
			}
		}`,
			shouldSucceed: false,
			wantErrTitle:  "Multiple targeting parameters.",
			wantErrDetail: ErrRequestIncludesMultipleTargetingParams.Error(),
		},
		{
			name: "error when group ids and tags included",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"group_ids": ["group-1"],
			"tags": {
				"tags": [
					"linux",
					"windows"
				],
				"operator": "OR"
			}
		}`,
			shouldSucceed: false,
			wantErrTitle:  "Multiple targeting parameters.",
			wantErrDetail: ErrRequestIncludesMultipleTargetingParams.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			curUser := makeTestUserWithToken(testUser, testToken)

			connMock1 := makeConnMock(t, true, 1, time.Date(2020, 10, 10, 10, 10, 1, 0, time.UTC))
			connMock2 := makeConnMock(t, true, 2, time.Date(2020, 10, 10, 10, 10, 2, 0, time.UTC))
			connMock4 := makeConnMock(t, true, 4, time.Date(2020, 10, 10, 10, 10, 4, 0, time.UTC))

			c1 := clients.New(t).ID("client-1").Connection(connMock1).Build()
			c2 := clients.New(t).ID("client-2").Connection(connMock2).Build()
			c3 := clients.New(t).ID("client-3").DisconnectedDuration(5 * time.Minute).Build()
			c4 := clients.New(t).ID("client-4").Connection(connMock4).Build()

			c1.Tags = []string{"linux"}
			c2.Tags = []string{"windows"}
			c3.Tags = []string{"mac"}
			c4.Tags = []string{"linux", "windows"}

			g1 := makeClientGroup("group-1", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-1", "client-2"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			g2 := makeClientGroup("group-2", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-4"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			c1.AllowedUserGroups = []string{"group-1"}
			c2.AllowedUserGroups = []string{"group-1"}
			c4.AllowedUserGroups = []string{"group-2"}

			clientList := []*clients.Client{c1, c2, c4}

			p := clients.NewFakeClientProvider(t, nil, nil)

			al := makeAPIListener(p,
				curUser,
				clients.NewClientRepositoryWithDB(nil, &hour, p, testLog),
				defaultTimeout,
				testLog)

			// make sure the repo has the test clients
			for _, cl := range clientList {
				err := al.clientService.GetRepo().Save(cl)
				assert.NoError(t, err)
			}

			var done chan bool
			if tc.shouldSucceed {
				done = make(chan bool)
				al.testDone = done
			}

			jp := makeJobsProvider(t, DataSourceOptions, testLog)
			defer jp.Close()

			gp := makeGroupsProvider(t, DataSourceOptions, testLog)
			defer gp.Close()

			al.initRouter()

			al.jobProvider = jp
			al.clientGroupProvider = gp

			ctx := api.WithUser(context.Background(), testUser)

			err := gp.Create(ctx, g1)
			assert.NoError(t, err)
			err = gp.Create(ctx, g2)
			assert.NoError(t, err)

			// setup a web socket server running the handler under test
			s := httptest.NewServer(al.wsAuth(http.HandlerFunc(al.handleCommandsWS)))
			defer s.Close()

			// prep the test user auth
			reqHeader := makeAuthHeader(testUser, testToken)

			// dial the test websocket server running the handler under test
			wsURL := httpToWS(t, s.URL)
			ws, _, err := websocket.DefaultDialer.Dial(wsURL, reqHeader)
			assert.NoError(t, err)
			defer ws.Close()

			// send the request to the handler under test
			err = ws.WriteMessage(websocket.TextMessage, []byte(tc.requestBody))
			assert.NoError(t, err)

			if tc.shouldSucceed {
				<-al.testDone

				multiJobIDs := al.jobsDoneChannel.GetAllKeys()
				multiJobID := multiJobIDs[0]
				multiJob, err := jp.GetMultiJob(ctx, multiJobID)
				assert.NoError(t, err)

				assert.Equal(t, tc.wantJobCount, len(multiJob.Jobs))

				// ask the server to close the web socket
				err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				assert.NoError(t, err)
			} else {
				_, res, err := ws.ReadMessage()
				assert.NoError(t, err)

				result := strings.TrimSpace(string(res))
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), result)
			}
		})
	}
}

func TestHandlePostMultiClientScriptWithTags(t *testing.T) {

	defaultTimeout := 60

	testCases := []struct {
		name string

		requestBody string

		wantStatusCode int
		wantJobCount   int
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
	}{
		{
			name: "valid when only tags included",
			requestBody: `{
				"script": "dGVzdC5zaA==",
				"timeout_sec": 30,
				"tags": {
					"tags": [
						"linux"
					],
					"operator": "OR"
				},
				"abort_on_error": false,
				"execute_concurrently": false
			}`,
			wantStatusCode: http.StatusOK,
			wantJobCount:   2,
		},
		{
			name: "no targeting params provided",
			requestBody: `
		{
			"script": "dGVzdC5zaA==",
			"timeout_sec": 30
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Missing targeting parameters.",
			wantErrDetail:  ErrRequestMissingTargetingParams.Error(),
		},
		{
			name: "error when empty tags",
			requestBody: `
		{
			"script": "dGVzdC5zaA==",
			"timeout_sec": 30,
			"tags": {
				"tags": [],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "No tags specified.",
			wantErrDetail:  ErrMissingTagsInMultiJobRequest.Error(),
		},
		{
			name: "error when no clients for tag",
			requestBody: `
		{
			"script": "dGVzdC5zaA==",
			"timeout_sec": 30,
			"tags": {
				"tags": ["random"],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "no clients to execute the script for",
		},
		{
			name: "error when client ids and tags included",
			requestBody: `
		{
			"script": "dGVzdC5zaA==",
			"timeout_sec": 30,
			"client_ids": ["client-1", "client-2"],
			"tags": {
				"tags": [
					"linux",
					"windows"
				],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Multiple targeting parameters.",
			wantErrDetail:  ErrRequestIncludesMultipleTargetingParams.Error(),
		},
		{
			name: "error when group ids and tags included",
			requestBody: `
		{
			"script": "dGVzdC5zaA==",
			"timeout_sec": 30,
			"group_ids": ["group-1"],
			"tags": {
				"tags": [
					"linux",
					"windows"
				],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "Multiple targeting parameters.",
			wantErrDetail:  ErrRequestIncludesMultipleTargetingParams.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given

			testUser := "test-user"
			curUser := makeTestUser(testUser)

			connMock1 := makeConnMock(t, true, 1, time.Date(2020, 10, 10, 10, 10, 1, 0, time.UTC))
			connMock2 := makeConnMock(t, true, 2, time.Date(2020, 10, 10, 10, 10, 2, 0, time.UTC))
			connMock4 := makeConnMock(t, true, 4, time.Date(2020, 10, 10, 10, 10, 4, 0, time.UTC))

			c1 := clients.New(t).ID("client-1").Connection(connMock1).Build()
			c2 := clients.New(t).ID("client-2").Connection(connMock2).Build()
			c3 := clients.New(t).ID("client-3").DisconnectedDuration(5 * time.Minute).Build()
			c4 := clients.New(t).ID("client-4").Connection(connMock4).Build()

			c1.Tags = []string{"linux"}
			c2.Tags = []string{"windows"}
			c3.Tags = []string{"mac"}
			c4.Tags = []string{"linux", "windows"}

			g1 := makeClientGroup("group-1", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-1", "client-2"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			g2 := makeClientGroup("group-2", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-4"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			c1.AllowedUserGroups = []string{"group-1"}
			c2.AllowedUserGroups = []string{"group-1"}
			c4.AllowedUserGroups = []string{"group-2"}

			clientList := []*clients.Client{c1, c2, c4}

			p := clients.NewFakeClientProvider(t, nil, nil)

			al := makeAPIListener(p,
				curUser,
				clients.NewClientRepositoryWithDB(nil, &hour, p, testLog),
				defaultTimeout,
				testLog)

			// make sure the repo has the test clients
			for _, cl := range clientList {
				err := al.clientService.GetRepo().Save(cl)
				assert.NoError(t, err)
			}

			var done chan bool
			if tc.wantStatusCode == http.StatusOK {
				done = make(chan bool)
				al.testDone = done
			}

			jp := makeJobsProvider(t, DataSourceOptions, testLog)
			defer jp.Close()

			gp := makeGroupsProvider(t, DataSourceOptions, testLog)
			defer gp.Close()

			al.initRouter()

			al.jobProvider = jp
			al.clientGroupProvider = gp

			ctx := api.WithUser(context.Background(), testUser)

			err := gp.Create(ctx, g1)
			assert.NoError(t, err)
			err = gp.Create(ctx, g2)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/scripts", strings.NewReader(tc.requestBody))
			req = req.WithContext(ctx)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantStatusCode == http.StatusOK {
				// wait until async task executeMultiClientJob finishes
				<-al.testDone
			}
			if tc.wantErrTitle == "" {
				// success case
				assert.Contains(t, w.Body.String(), `{"data":{"jid":`)
				gotResp := api.NewSuccessPayload(newJobResponse{})
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotResp))
				gotPropMap, ok := gotResp.Data.(map[string]interface{})
				require.True(t, ok)
				jidObj, found := gotPropMap["jid"]
				require.True(t, found)
				gotJID, ok := jidObj.(string)
				require.True(t, ok)
				require.NotEmpty(t, gotJID)

				gotMultiJob, err := jp.GetMultiJob(ctx, gotJID)
				require.NoError(t, err)
				require.NotNil(t, gotMultiJob)
				require.Len(t, gotMultiJob.Jobs, tc.wantJobCount)
			} else {
				// failure case
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), w.Body.String())
			}
		})
	}
}

func TestHandlePostMultiClientWSScriptWithTags(t *testing.T) {
	defaultTimeout := 60

	testCases := []struct {
		name string

		requestBody string

		shouldSucceed bool
		wantJobCount  int
		wantErrCode   string
		wantErrTitle  string
		wantErrDetail string
	}{
		{
			name: "valid with client ids",
			requestBody: `
		{
			"script": "dGVzdC5zaA==",
			"timeout_sec": 30,
			"client_ids": ["client-1", "client-2"],
			"abort_on_error": false,
			"execute_concurrently": false
		}`,
			shouldSucceed: true,
			wantJobCount:  2,
		},
		{
			name: "no targeting params provided",
			requestBody: `
		{
			"script": "dGVzdC5zaA==",
			"timeout_sec": 30
		}`,
			shouldSucceed: false,
			wantErrTitle:  "Missing targeting parameters.",
			wantErrDetail: ErrRequestMissingTargetingParams.Error(),
		},
		{
			name: "valid when only tags included",
			requestBody: `{
				"script": "dGVzdC5zaA==",
				"timeout_sec": 30,
				"tags": {
					"tags": [
						"linux",
						"windows"
					],
					"operator": "AND"
				},
				"abort_on_error": false,
				"execute_concurrently": false
			}`,
			shouldSucceed: true,
			wantJobCount:  1,
		},
		{
			name: "error when empty tags",
			requestBody: `
		{
			"script": "dGVzdC5zaA==",
			"timeout_sec": 30,
			"tags": {
				"tags": [],
				"operator": "OR"
			}
		}`,
			shouldSucceed: false,
			wantErrTitle:  "No tags specified.",
			wantErrDetail: ErrMissingTagsInMultiJobRequest.Error(),
		},
		{
			name: "error when no clients for tag",
			requestBody: `
		{
			"script": "dGVzdC5zaA==",
			"timeout_sec": 30,
			"tags": {
				"tags": ["random"],
				"operator": "OR"
			}
		}`,
			shouldSucceed: false,
			wantErrTitle:  "no clients to execute the script for",
		},
		{
			name: "error when client ids and tags included",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"client_ids": ["client-1", "client-2"],
			"tags": {
				"tags": [
					"linux",
					"windows"
				],
				"operator": "OR"
			}
		}`,
			shouldSucceed: false,
			wantErrTitle:  "Multiple targeting parameters.",
			wantErrDetail: ErrRequestIncludesMultipleTargetingParams.Error(),
		},
		{
			name: "error when group ids and tags included",
			requestBody: `
		{
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"group_ids": ["group-1"],
			"tags": {
				"tags": [
					"linux",
					"windows"
				],
				"operator": "OR"
			}
		}`,
			shouldSucceed: false,
			wantErrTitle:  "Multiple targeting parameters.",
			wantErrDetail: ErrRequestIncludesMultipleTargetingParams.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given

			testUser := "test-user"
			testToken := "12345678"
			curUser := makeTestUserWithToken(testUser, testToken)

			connMock1 := makeConnMock(t, true, 1, time.Date(2020, 10, 10, 10, 10, 1, 0, time.UTC))
			connMock2 := makeConnMock(t, true, 2, time.Date(2020, 10, 10, 10, 10, 2, 0, time.UTC))
			connMock4 := makeConnMock(t, true, 4, time.Date(2020, 10, 10, 10, 10, 4, 0, time.UTC))

			c1 := clients.New(t).ID("client-1").Connection(connMock1).Build()
			c2 := clients.New(t).ID("client-2").Connection(connMock2).Build()
			c3 := clients.New(t).ID("client-3").DisconnectedDuration(5 * time.Minute).Build()
			c4 := clients.New(t).ID("client-4").Connection(connMock4).Build()

			c1.Tags = []string{"linux"}
			c2.Tags = []string{"windows"}
			c3.Tags = []string{"mac"}
			c4.Tags = []string{"linux", "windows"}

			g1 := makeClientGroup("group-1", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-1", "client-2"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			g2 := makeClientGroup("group-2", &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{"client-4"},
				OS:       &cgroups.ParamValues{"Linux*"},
				Version:  &cgroups.ParamValues{"0.1.1*"},
			})

			c1.AllowedUserGroups = []string{"group-1"}
			c2.AllowedUserGroups = []string{"group-1"}
			c4.AllowedUserGroups = []string{"group-2"}

			clientList := []*clients.Client{c1, c2, c4}

			p := clients.NewFakeClientProvider(t, nil, nil)

			al := makeAPIListener(p,
				curUser,
				clients.NewClientRepositoryWithDB(nil, &hour, p, testLog),
				defaultTimeout,
				testLog)

			// make sure the repo has the test clients
			for _, cl := range clientList {
				err := al.clientService.GetRepo().Save(cl)
				assert.NoError(t, err)
			}

			var done chan bool
			if tc.shouldSucceed {
				done = make(chan bool)
				al.testDone = done
			}

			jp := makeJobsProvider(t, DataSourceOptions, testLog)
			defer jp.Close()

			gp := makeGroupsProvider(t, DataSourceOptions, testLog)
			defer gp.Close()

			al.initRouter()

			al.jobProvider = jp
			al.clientGroupProvider = gp

			ctx := api.WithUser(context.Background(), testUser)

			err := gp.Create(ctx, g1)
			assert.NoError(t, err)
			err = gp.Create(ctx, g2)
			assert.NoError(t, err)

			// setup a web socket server running the handler under test
			s := httptest.NewServer(al.wsAuth(http.HandlerFunc(al.handleScriptsWS)))
			defer s.Close()

			// prep the test user auth
			reqHeader := makeAuthHeader(testUser, testToken)

			// dial the test websocket server running the handler under test
			wsURL := httpToWS(t, s.URL)
			ws, _, err := websocket.DefaultDialer.Dial(wsURL, reqHeader)
			assert.NoError(t, err)
			defer ws.Close()

			// send the request to the handler under test
			err = ws.WriteMessage(websocket.TextMessage, []byte(tc.requestBody))
			assert.NoError(t, err)

			if tc.shouldSucceed {
				<-al.testDone

				multiJobIDs := al.jobsDoneChannel.GetAllKeys()
				multiJobID := multiJobIDs[0]
				multiJob, err := jp.GetMultiJob(ctx, multiJobID)
				assert.NoError(t, err)

				assert.Equal(t, tc.wantJobCount, len(multiJob.Jobs))

				// ask the server to close the web socket
				err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				assert.NoError(t, err)
			} else {
				_, res, err := ws.ReadMessage()
				assert.NoError(t, err)

				result := strings.TrimSpace(string(res))
				wantResp := api.NewErrAPIPayloadFromMessage(tc.wantErrCode, tc.wantErrTitle, tc.wantErrDetail)
				wantRespBytes, err := json.Marshal(wantResp)
				require.NoError(t, err)
				require.Equal(t, string(wantRespBytes), result)
			}
		})
	}
}

func makeTestUser(testUser string) (curUser *users.User) {
	curUser = &users.User{
		Username: testUser,
		Groups:   []string{users.Administrators},
	}
	return curUser
}

func makeTestUserWithToken(testUser string, token string) (curUser *users.User) {
	curUser = &users.User{
		Username: testUser,
		Groups:   []string{users.Administrators},
		Token:    &token,
	}
	return curUser
}

func makeAuthHeader(testUser string, testToken string) (reqHeader http.Header) {
	auth := testUser + ":" + testToken
	authContent := base64.StdEncoding.EncodeToString([]byte(auth))
	reqHeader = http.Header{}
	reqHeader.Add("Authorization", "Basic "+authContent)
	return reqHeader
}

func makeClientGroup(groupID string, params *cgroups.ClientParams) (gp *cgroups.ClientGroup) {
	gp = &cgroups.ClientGroup{
		ID:     groupID,
		Params: params,
	}
	return gp
}

func httpToWS(t *testing.T, u string) string {
	t.Helper()

	wsURL, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}

	switch wsURL.Scheme {
	case "http":
		wsURL.Scheme = "ws"
	case "https":
		wsURL.Scheme = "wss"
	}

	return wsURL.String()
}

func makeConnMock(t *testing.T, returnOk bool, pid int, startedAt time.Time) (connMock *test.ConnMock) {
	t.Helper()
	connMock = test.NewConnMock()
	connMock.ReturnOk = returnOk
	sshSuccessResp := comm.RunCmdResponse{Pid: pid, StartedAt: startedAt}
	sshRespBytes, err := json.Marshal(sshSuccessResp)
	require.NoError(t, err)
	connMock.ReturnResponsePayload = sshRespBytes
	return connMock
}

func makeAPIListener(
	p *clients.SqliteProvider,
	curUser *users.User,
	clientRepo *clients.ClientRepository,
	defaultTimeout int,
	testLog *logger.Logger) (al *APIListener) {
	al = &APIListener{
		insecureForTests: true,
		Server: &Server{
			clientService: NewClientService(nil, nil, clientRepo),
			config: &Config{
				Server: ServerConfig{
					RunRemoteCmdTimeoutSec: defaultTimeout,
					MaxRequestBytes:        1024 * 1024,
				},
			},
			uiJobWebSockets: ws.NewWebSocketCache(),
			jobsDoneChannel: jobResultChanMap{
				m: make(map[string]chan *models.Job),
			},
		},
		bannedUsers: security.NewBanList(time.Duration(60) * time.Second),
		userService: users.NewAPIService(users.NewStaticProvider([]*users.User{curUser}), false),
		Logger:      testLog,
	}

	return al
}

func makeJobsProvider(t *testing.T, dataSourceOptions sqlite.DataSourceOptions, testLog *logger.Logger) (jp *jobs.SqliteProvider) {
	t.Helper()
	jobsDB, err := sqlite.New(
		":memory:",
		jobsmigration.AssetNames(),
		jobsmigration.Asset,
		DataSourceOptions,
	)
	require.NoError(t, err)
	jp = jobs.NewSqliteProvider(jobsDB, testLog)
	return jp
}

func makeGroupsProvider(t *testing.T, dataSourceOptions sqlite.DataSourceOptions, testLog *logger.Logger) (gp *cgroups.SqliteProvider) {
	groupsDB, err := sqlite.New(
		":memory:",
		client_groups.AssetNames(),
		client_groups.Asset,
		DataSourceOptions,
	)
	require.NoError(t, err)

	gp, err = cgroups.NewSqliteProvider(groupsDB)
	assert.NoError(t, err)
	return gp
}

func makeScheduleManager(t *testing.T, jp *jobs.SqliteProvider, jobRunner schedule.JobRunner, dataSourceOptions sqlite.DataSourceOptions, testLog *logger.Logger) (scheduleManager *schedule.Manager) {
	scheduleManager = schedule.NewManager(jobRunner, jp.GetDB(), testLog, 30)

	return scheduleManager
}
