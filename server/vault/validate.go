package vault

import (
	"net/http"

	errors2 "github.com/openrport/openrport/server/api/errors"
)

func Validate(iv *InputValue) error {
	errs := errors2.APIErrors{}

	if iv.Key == "" {
		errs = append(errs, errors2.APIError{
			Message:    "key is required",
			HTTPStatus: http.StatusBadRequest,
		})
	}
	if iv.Value == "" {
		errs = append(errs, errors2.APIError{
			Message:    "value is required",
			HTTPStatus: http.StatusBadRequest,
		})
	}

	if iv.Type == "" {
		errs = append(errs, errors2.APIError{
			Message:    "value type is required",
			HTTPStatus: http.StatusBadRequest,
		})
	} else {
		knownTypes := map[ValueType]bool{
			TextType:     true,
			SecretType:   true,
			MarkdownType: true,
			StringType:   true,
		}

		ok := knownTypes[iv.Type]
		if !ok {
			errs = append(errs, errors2.APIError{
				Message:    "unknown or invalid value value type " + string(iv.Type),
				HTTPStatus: http.StatusBadRequest,
			})
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}
