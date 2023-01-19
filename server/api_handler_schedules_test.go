package chserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/jobs/schedule"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
)

func TestHandlePostScheduleMultiClientJobWithTags(t *testing.T) {
	testUser := "test-user"
	defaultTimeout := 60

	testCases := []struct {
		name string

		requestBody string

		wantStatusCode int
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
	}{
		{
			name: "valid when only tags included",
			requestBody: `{
				"type": "command",
				"schedule": "* * * * *",
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
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "valid when only tags included and missing operator",
			requestBody: `{
				"type": "command",
				"schedule": "* * * * *",
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
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "error when client ids and tags included",
			requestBody: `
		{
			"type": "command",
			"schedule": "* * * * *",
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
			"type": "command",
			"schedule": "* * * * *",
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
			"type": "command",
			"schedule": "* * * * *",
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"tags": {
				"tags": ["random"],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "at least 1 client should be specified",
		},
		{
			name: "error when group ids and tags included",
			requestBody: `
		{
			"type": "command",
			"schedule": "* * * * *",
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

			connMock1 := makeConnMock(t, 1, time.Date(2020, 10, 10, 10, 10, 1, 0, time.UTC))
			connMock2 := makeConnMock(t, 2, time.Date(2020, 10, 10, 10, 10, 2, 0, time.UTC))
			connMock4 := makeConnMock(t, 4, time.Date(2020, 10, 10, 10, 10, 4, 0, time.UTC))

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

			al := makeAPIListener(curUser,
				clients.NewClientRepositoryWithDB(nil, &hour, p, testLog),
				defaultTimeout,
				nil,
				testLog)

			// make sure the repo has the test clients
			for _, cl := range clientList {
				err := al.clientService.GetRepo().Save(cl)
				assert.NoError(t, err)
			}

			jp := makeJobsProvider(t, DataSourceOptions, testLog)
			defer jp.Close()

			gp := makeGroupsProvider(t, DataSourceOptions)
			defer gp.Close()

			scheduleManager := makeScheduleManager(t, jp, al, testLog)

			al.initRouter()

			al.jobProvider = jp
			al.clientGroupProvider = gp
			al.scheduleManager = scheduleManager

			ctx := api.WithUser(context.Background(), testUser)

			err := gp.Create(ctx, g1)
			assert.NoError(t, err)
			err = gp.Create(ctx, g2)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", strings.NewReader(tc.requestBody))
			req = req.WithContext(ctx)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantStatusCode == http.StatusCreated {
				// success case
				assert.Contains(t, w.Body.String(), `{"data":{"id":`)
				gotResp := api.NewSuccessPayload(newJobResponse{})
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotResp))
				gotPropMap, ok := gotResp.Data.(map[string]interface{})
				require.True(t, ok)
				idObj, found := gotPropMap["id"]
				require.True(t, found)
				gotID, ok := idObj.(string)
				require.True(t, ok)
				require.NotEmpty(t, gotID)
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

func TestHandlePostUpdateScheduleMultiClientJobWithTags(t *testing.T) {
	testUser := "test-user"
	defaultTimeout := 60

	testCases := []struct {
		name string

		requestBody string

		wantStatusCode int
		wantErrCode    string
		wantErrTitle   string
		wantErrDetail  string
	}{
		{
			name: "valid when only tags included",
			requestBody: `{
				"type": "command",
				"schedule": "* * * * *",
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
		},
		{
			name: "valid when only tags included and missing operator",
			requestBody: `{
				"type": "command",
				"schedule": "* * * * *",
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
		},
		{
			name: "error when client ids and tags included",
			requestBody: `
		{
			"type": "command",
			"schedule": "* * * * *",
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
			"type": "command",
			"schedule": "* * * * *",
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
			"type": "command",
			"schedule": "* * * * *",
			"command": "/bin/date;foo;whoami",
			"timeout_sec": 30,
			"tags": {
				"tags": ["random"],
				"operator": "OR"
			}
		}`,
			wantStatusCode: http.StatusBadRequest,
			wantErrTitle:   "at least 1 client should be specified",
		},
		{
			name: "error when group ids and tags included",
			requestBody: `
		{
			"type": "command",
			"schedule": "* * * * *",
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

			connMock1 := makeConnMock(t, 1, time.Date(2020, 10, 10, 10, 10, 1, 0, time.UTC))
			connMock2 := makeConnMock(t, 2, time.Date(2020, 10, 10, 10, 10, 2, 0, time.UTC))
			connMock4 := makeConnMock(t, 4, time.Date(2020, 10, 10, 10, 10, 4, 0, time.UTC))

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

			al := makeAPIListener(curUser,
				clients.NewClientRepositoryWithDB(nil, &hour, p, testLog),
				defaultTimeout,
				nil,
				testLog)

			// make sure the repo has the test clients
			for _, cl := range clientList {
				err := al.clientService.GetRepo().Save(cl)
				assert.NoError(t, err)
			}

			jp := makeJobsProvider(t, DataSourceOptions, testLog)
			defer jp.Close()

			gp := makeGroupsProvider(t, DataSourceOptions)
			defer gp.Close()

			scheduleManager := makeScheduleManager(t, jp, al, testLog)

			al.initRouter()

			al.jobProvider = jp
			al.clientGroupProvider = gp
			al.scheduleManager = scheduleManager

			ctx := api.WithUser(context.Background(), testUser)

			err := gp.Create(ctx, g1)
			assert.NoError(t, err)
			err = gp.Create(ctx, g2)
			assert.NoError(t, err)

			existingSchedule := &schedule.Schedule{
				Base: schedule.Base{
					Type:     schedule.TypeCommand,
					Schedule: "* * * * *",
				},
				Details: schedule.Details{
					ClientIDs: []string{"id-1", "id-2"},
					Command:   "/bin/true",
				},
			}
			existingSchedule, err = scheduleManager.Create(ctx, existingSchedule, testUser)
			assert.NoError(t, err)

			updateURLPath := fmt.Sprintf("/api/v1/schedules/%s", existingSchedule.ID)

			req := httptest.NewRequest(http.MethodPut, updateURLPath, strings.NewReader(tc.requestBody))
			req = req.WithContext(ctx)

			// when
			w := httptest.NewRecorder()
			al.router.ServeHTTP(w, req)

			// then
			assert.Equal(t, tc.wantStatusCode, w.Code)
			if tc.wantStatusCode == http.StatusOK {
				// success case
				assert.Contains(t, w.Body.String(), `{"data":{"id":`)
				gotResp := api.NewSuccessPayload(newJobResponse{})
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotResp))
				gotPropMap, ok := gotResp.Data.(map[string]interface{})
				require.True(t, ok)
				idObj, found := gotPropMap["id"]
				require.True(t, found)
				gotID, ok := idObj.(string)
				require.True(t, ok)
				require.NotEmpty(t, gotID)
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
