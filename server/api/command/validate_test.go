package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name          string
		input         *InputCommand
		expectedError string
	}{
		{
			name:          "missing name and command",
			input:         &InputCommand{},
			expectedError: "name is required, cmd is required",
		}, {
			name: "missing name",
			input: &InputCommand{
				Cmd: "val1",
			},
			expectedError: "name is required",
		}, {
			name: "missing command",
			input: &InputCommand{
				Name: "some name",
			},
			expectedError: "cmd is required",
		}, {
			name: "all ok",
			input: &InputCommand{
				Name: "some name",
				Cmd:  "val1",
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
