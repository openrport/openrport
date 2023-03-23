package schedule

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/server/api/jobs"
	"github.com/realvnc-labs/rport/share/models"
	"github.com/realvnc-labs/rport/share/ptr"
)

func TestToLastExecution(t *testing.T) {
	testTime := time.Date(2022, 5, 10, 14, 12, 50, 0, time.UTC)
	testCases := []struct {
		Name     string
		Input    Execution
		Expected string
	}{
		{
			Name: "no data",
			Input: Execution{
				StartedAt: nil,
			},
			Expected: "null",
		},
		{
			Name: "single client",
			Input: Execution{
				StartedAt:    &testTime,
				ClientCount:  ptr.Int(1),
				SuccessCount: ptr.Int(1),
				Status:       ptr.String(models.JobStatusSuccessful),
				Details: &jobs.JobDetails{
					Result: &models.JobResult{
						Summary: "all ok",
					},
				},
			},
			Expected: `{
				"started_at": "2022-05-10T14:12:50Z",
				"client_count": 1,
				"success_count": 1,
				"status": "successful",
				"summary": "all ok"
			}`,
		},
		{
			Name: "multiple clients",
			Input: Execution{
				StartedAt:    &testTime,
				ClientCount:  ptr.Int(3),
				SuccessCount: ptr.Int(2),
				Status:       ptr.String(models.JobStatusSuccessful),
				Details: &jobs.JobDetails{
					Result: &models.JobResult{
						Summary: "all ok",
					},
				},
			},
			Expected: `{
				"started_at": "2022-05-10T14:12:50Z",
				"client_count": 3,
				"success_count": 2,
				"status": null,
				"summary": null
			}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			result := tc.Input.ToLastExecution()
			resultJSON, err := json.Marshal(result)
			require.NoError(t, err)

			assert.JSONEq(t, tc.Expected, string(resultJSON))
		})
	}
}
