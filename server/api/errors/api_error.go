package errors

import "strings"

// APIError wraps error which is interpreted as in http error
type APIError struct {
	Message string
	Err     error
	Code    int
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
