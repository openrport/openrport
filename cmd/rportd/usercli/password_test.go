package usercli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptForPassword(t *testing.T) {
	testCases := []struct {
		Name        string
		Password1   string
		Password2   string
		Expected    string
		ExpectedErr error
	}{
		{
			Name:      "ok",
			Password1: "testpassword",
			Password2: "testpassword",
			Expected:  "testpassword",
		},
		{
			Name:        "different",
			Password1:   "testpassword",
			Password2:   "different",
			ExpectedErr: ErrPasswordsDoNotMatch,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			mockPasswordReader := &passwordReaderMock{
				Passwords: []string{tc.Password1, tc.Password2},
			}

			result, err := promptForPassword(mockPasswordReader)

			assert.Equal(t, 2, mockPasswordReader.CallCount)
			if tc.ExpectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, tc.Expected, result)
			} else {
				assert.ErrorIs(t, err, tc.ExpectedErr)
			}
		})
	}
}

type passwordReaderMock struct {
	Passwords []string
	CallCount int
}

func (m *passwordReaderMock) ReadPassword() (string, error) {
	defer func() {
		m.CallCount++
	}()
	return m.Passwords[m.CallCount], nil
}
