package vault

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/share/enc"
)

func TestInvalidPasswords(t *testing.T) {
	testCases := []struct {
		InputPass     string
		ExpectedError string
		ExpectedCode  int
	}{
		{
			InputPass:     "1",
			ExpectedError: "password is too short, expected length is 4 bytes, provided length is 1 bytes",
			ExpectedCode:  http.StatusBadRequest,
		},
		{
			InputPass:     "12",
			ExpectedError: "password is too short, expected length is 4 bytes, provided length is 2 bytes",
			ExpectedCode:  http.StatusBadRequest,
		},
		{
			InputPass:     "123",
			ExpectedError: "password is too short, expected length is 4 bytes, provided length is 3 bytes",
			ExpectedCode:  http.StatusBadRequest,
		},
		{
			InputPass:     "123456789012345678901234567890123",
			ExpectedError: "password is too long, expected length is 32 bytes, provided length is 33 bytes",
			ExpectedCode:  http.StatusBadRequest,
		},
	}

	for i := range testCases {
		passManager := Aes256PassManager{}
		err := passManager.ValidatePass(testCases[i].InputPass)
		apiErr, ok := err.(errors.APIError)
		require.True(t, ok)
		assert.Equal(t, testCases[i].ExpectedError, apiErr.Message)
		assert.Equal(t, testCases[i].ExpectedCode, apiErr.HTTPStatus)
	}
}

func TestValidPassword(t *testing.T) {
	passManager := Aes256PassManager{}
	err := passManager.ValidatePass("1234")
	require.NoError(t, err)
}

func TestPassMatch(t *testing.T) {
	passManager := Aes256PassManager{}

	const passToGive = "1234"
	const dataToGive = "some enc value"
	encPass, err := enc.Aes256EncryptByPassToBase64String([]byte(dataToGive), passToGive)
	require.NoError(t, err)

	inputDbStatus := DbStatus{
		EncCheckValue: encPass,
		DecCheckValue: dataToGive,
	}

	isMatched, err := passManager.PassMatch(inputDbStatus, passToGive)
	require.NoError(t, err)
	assert.True(t, isMatched)

	isMatched2, err2 := passManager.PassMatch(inputDbStatus, passToGive+"123")
	require.NoError(t, err2)
	assert.False(t, isMatched2)

	inputDbStatus.EncCheckValue += "123"
	isMatched3, err3 := passManager.PassMatch(inputDbStatus, passToGive)
	require.NoError(t, err3)
	assert.False(t, isMatched3)
}

func TestPassMatchWithInvalidInput(t *testing.T) {
	passManager := Aes256PassManager{}
	inputDbStatus := DbStatus{
		EncCheckValue: "123",
	}

	isMatched, err := passManager.PassMatch(inputDbStatus, "")
	require.NoError(t, err)
	assert.False(t, isMatched)

	_, err2 := passManager.PassMatch(DbStatus{}, "123")
	require.EqualError(t, err2, "password control value from db is empty")
}

func TestGetEncRandValue(t *testing.T) {
	passManager := Aes256PassManager{}
	const passToGive = "1234"

	actualEncValue, actualDecValue, err := passManager.GetEncRandValue(passToGive)
	require.NoError(t, err)

	expectedDecValue, err := enc.Aes256DecryptByPassFromBase64String(actualEncValue, passToGive)
	require.NoError(t, err)
	assert.Equal(t, string(expectedDecValue), actualDecValue)
}
