package vault

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name          string
		input         *InputValue
		expectedError string
	}{
		{
			name: "missing_key_value_type",
			input: &InputValue{
				Key:   "",
				Value: "",
				Type:  "",
			},
			expectedError: "key is required, value is required, value type is required",
		},
		{
			name: "all_ok0",
			input: &InputValue{
				Key:   "k1",
				Value: "val1",
				Type:  TextType,
			},
			expectedError: "",
		},
		{
			name: "all_ok1",
			input: &InputValue{
				Key:   "k1",
				Value: "val1",
				Type:  SecretType,
			},
			expectedError: "",
		},
		{
			name: "all_ok2",
			input: &InputValue{
				Key:   "k1",
				Value: "val1",
				Type:  MarkdownType,
			},
			expectedError: "",
		},
		{
			name: "invalid_value_type",
			input: &InputValue{
				Key:   "k1",
				Value: "val1",
				Type:  "some type",
			},
			expectedError: "unknown or invalid value value type some type",
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
