package command

import (
	"net/http"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

func Validate(iv *InputCommand) error {
	errs := errors2.APIErrors{}

	if iv.Name == "" {
		errs = append(errs, errors2.APIError{
			Message:    "name is required",
			HTTPStatus: http.StatusBadRequest,
		})
	}
	if iv.Cmd == "" {
		errs = append(errs, errors2.APIError{
			Message:    "cmd is required",
			HTTPStatus: http.StatusBadRequest,
		})
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}
