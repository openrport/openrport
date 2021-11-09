package query

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cloudradar-monitoring/rport/server/api/errors"
	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

type Pagination struct {
	Limit  string
	Offset string
}

type PaginationConfig struct {
	MaxLimit     int
	DefaultLimit int
}

func ValidatePagination(pagination *Pagination, config *PaginationConfig) errors2.APIErrors {
	errs := errors2.APIErrors{}

	if pagination.Limit == "" {
		pagination.Limit = strconv.Itoa(config.DefaultLimit)
	}

	limit, err := strconv.Atoi(pagination.Limit)
	if err != nil {
		errs = append(errs, errors.APIError{
			Message:    "pagination limit must be a number",
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		})
	} else {
		if limit > config.MaxLimit {
			errs = append(errs, errors2.APIError{
				Message:    fmt.Sprintf("pagination limit too big (%v) maximum is %v", pagination.Limit, config.MaxLimit),
				HTTPStatus: http.StatusBadRequest,
			})
		}
		if limit <= 0 {
			errs = append(errs, errors2.APIError{
				Message:    "pagination limit must be positive",
				HTTPStatus: http.StatusBadRequest,
			})
		}
	}

	offset, err := strconv.Atoi(pagination.Offset)
	if err != nil {
		errs = append(errs, errors.APIError{
			Message:    "pagination offset must be a number",
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		})
	} else {
		if offset < 0 {
			errs = append(errs, errors2.APIError{
				Message:    "pagination offset must not be negative",
				HTTPStatus: http.StatusBadRequest,
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func ParsePagination(values url.Values) *Pagination {
	p := &Pagination{
		Offset: "0",
	}

	limit, ok := values["page[limit]"]
	if ok && len(limit) > 0 {
		p.Limit = limit[0]
	}

	offset, ok := values["page[offset]"]
	if ok && len(offset) > 0 {
		p.Offset = offset[0]
	}

	return p
}
