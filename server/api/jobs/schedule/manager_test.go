package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	manager := &Manager{
		cron: newCron(),
	}

	testCases := []struct {
		Name          string
		Schedule      *Schedule
		ExpectedError string
	}{
		{
			Name: "invalid type",
			Schedule: &Schedule{
				Base: Base{
					Type: "invalid",
				},
			},
			ExpectedError: "type must be 'command' or 'script'",
		},
		{
			Name: "invalid schedule",
			Schedule: &Schedule{
				Base: Base{
					Type:     TypeCommand,
					Schedule: "* * *",
				},
			},
			ExpectedError: "expected exactly 5 fields, found 3: [* * *]",
		},
		{
			Name: "empty command",
			Schedule: &Schedule{
				Base: Base{
					Type:     TypeCommand,
					Schedule: "* * * * *",
				},
				Details: Details{
					ClientIDs: []string{"id-1"},
				},
			},
			ExpectedError: "command cannot be empty",
		},
		{
			Name: "ok command",
			Schedule: &Schedule{
				Base: Base{
					Type:     TypeCommand,
					Schedule: "* * * * *",
				},
				Details: Details{
					ClientIDs: []string{"id-1"},
					Command:   "/bin/true",
				},
			},
			ExpectedError: "",
		},
		{
			Name: "empty script",
			Schedule: &Schedule{
				Base: Base{
					Type:     TypeScript,
					Schedule: "* * * * *",
				},
				Details: Details{
					ClientIDs: []string{"id-1"},
				},
			},
			ExpectedError: "script cannot be empty",
		},
		{
			Name: "invalid script",
			Schedule: &Schedule{
				Base: Base{
					Type:     TypeScript,
					Schedule: "* * * * *",
				},
				Details: Details{
					ClientIDs: []string{"id-1"},
					Script:    "invalid",
				},
			},
			ExpectedError: "illegal base64 data at input byte 4",
		},
		{
			Name: "ok script",
			Schedule: &Schedule{
				Base: Base{
					Type:     TypeScript,
					Schedule: "* * * * *",
				},
				Details: Details{
					GroupIDs: []string{"id-1"},
					Script:   "ZWNobyAndGVzdCc=",
				},
			},
			ExpectedError: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			err := manager.validate(tc.Schedule)

			if tc.ExpectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tc.ExpectedError, err.Error())
			}
		})
	}
}
