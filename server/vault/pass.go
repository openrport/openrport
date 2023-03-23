package vault

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	errors2 "github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/share/enc"
)

const (
	minPassLengthBytes = 4
	maxPassLengthBytes = 32
)

type Aes256PassManager struct {
}

func (apm *Aes256PassManager) ValidatePass(passToCheck string) error {
	if len(passToCheck) < minPassLengthBytes {
		return errors2.APIError{
			Message: fmt.Sprintf(
				"password is too short, expected length is %d bytes, provided length is %d bytes",
				minPassLengthBytes,
				len(passToCheck),
			),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	if len(passToCheck) > maxPassLengthBytes {
		return errors2.APIError{
			Message: fmt.Sprintf(
				"password is too long, expected length is %d bytes, provided length is %d bytes",
				maxPassLengthBytes,
				len(passToCheck),
			),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	return nil
}

func (apm *Aes256PassManager) PassMatch(dbStatus DbStatus, passToCheck string) (bool, error) {
	if passToCheck == "" {
		return false, nil
	}

	if dbStatus.EncCheckValue == "" {
		return false, errors.New("password control value from db is empty")
	}

	decValue, err := enc.Aes256DecryptByPassFromBase64String(dbStatus.EncCheckValue, passToCheck)

	subtle.ConstantTimeCompare(decValue, decValue) //to simulate constant time for password check

	if err != nil {
		return false, nil
	}

	return true, nil
}

// GetEncRandValue generates a pseudo random hash sum and encrypts it with the provided password
// this is used to check if the provided password is correct and can potentially decrypt vault values
func (apm *Aes256PassManager) GetEncRandValue(pass string) (encValue, decValue string, err error) {
	timestampStr := strconv.FormatInt(time.Now().UnixNano(), 10)
	timestampHash := sha256.New().Sum([]byte(timestampStr))

	encValue, err = enc.Aes256EncryptByPassToBase64String(timestampHash, pass)
	if err != nil {
		return encValue, decValue, err
	}

	return encValue, string(timestampHash), nil
}
