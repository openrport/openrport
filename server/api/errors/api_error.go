package errors

import (
	"strings"
)

// APIError wraps error which is interpreted as in http error
type APIError struct {
	Message    string
	Err        error
	HTTPStatus int
	ErrCode    string
}

func NewAPIError(statusCode int, errCode string, message string, err error) (ae APIError) {
	ae = APIError{
		HTTPStatus: statusCode,
		ErrCode:    errCode,
		Message:    message,
		Err:        err,
	}
	return ae
}

// Error interface implementation
func (ae APIError) Error() string {
	if ae.Err != nil {
		return ae.Err.Error()
	}

	return ae.Message
}

type APIErrors []APIError

func (aes APIErrors) Error() string {
	errsFlat := make([]string, 0, len(aes))
	for i := range aes {
		errsFlat = append(errsFlat, aes[i].Error())
	}

	return strings.Join(errsFlat, ", ")
}
