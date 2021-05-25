package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name          string
		input         *InputScript
		expectedError string
	}{
		{
			name:          "missing_name_and_script",
			input:         &InputScript{},
			expectedError: "name is required, script is required",
		},
		{
			name: "all_ok0",
			input: &InputScript{
				Name:   "some name",
				Script: "val1",
			},
			expectedError: "",
		},
		{
			name: "all_ok1",
			input: &InputScript{
				Name:        "some name 2",
				Interpreter: "bash",
				IsSudo:      false,
				Cwd:         "/root",
				Script:      "pwd",
			},
			expectedError: "",
		},
	}

	for i := range testCases {
		t.Run(testCases[i].name, func(t *testing.T) {
			err := Validate(testCases[i].input)
			if testCases[i].expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, testCases[i].expectedError)
			}
		})
	}
}
