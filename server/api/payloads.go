package api

import (
	"errors"

	errors2 "github.com/openrport/openrport/server/api/errors"
)

// SuccessPayload represents a uniform format for all successful API responses.
type SuccessPayload struct {
	Data  interface{} `json:"data"`
	Meta  *Meta       `json:"meta,omitempty"`
	Links Links       `json:"links,omitempty"`
}

func NewSuccessPayload(data interface{}) SuccessPayload {
	return SuccessPayload{
		Data: data,
	}
}

// ErrorPayload represents a uniform format for all error API responses.
type ErrorPayload struct {
	Errors []ErrorPayloadItem `json:"errors"`
}

// ErrorPayloadItem represents a uniform format for a single error used in API responses.
type ErrorPayloadItem struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func NewErrAPIPayloadFromMessage(code, title, detail string) ErrorPayload {
	return ErrorPayload{
		Errors: []ErrorPayloadItem{
			{
				Code:   code,
				Title:  title,
				Detail: detail,
			},
		},
	}
}

func NewErrAPIPayloadFromError(err error, code, detail string) ErrorPayload {
	var apiErr errors2.APIError
	var apiErrs errors2.APIErrors
	switch {
	case errors.As(err, &apiErr):
		return newAPIErrorPayload(apiErr)
	case errors.As(err, &apiErrs):
		return newAPIErrorsPayload(apiErrs)
	}

	return NewErrAPIPayloadFromMessage(code, err.Error(), detail)
}

func newAPIErrorPayload(err errors2.APIError) ErrorPayload {
	return ErrorPayload{
		Errors: []ErrorPayloadItem{newAPIErrorPayloadItem(err)},
	}
}

func newAPIErrorPayloadItem(err errors2.APIError) ErrorPayloadItem {
	if err.Err != nil && err.Message != "" {
		return ErrorPayloadItem{
			Title:  err.Message,
			Detail: err.Err.Error(),
		}
	}
	return ErrorPayloadItem{
		Title:  err.Error(),
		Detail: "",
	}
}

func newAPIErrorsPayload(errors errors2.APIErrors) ErrorPayload {
	ep := ErrorPayload{
		Errors: make([]ErrorPayloadItem, 0, len(errors)),
	}
	for i := range errors {
		ep.Errors = append(ep.Errors, newAPIErrorPayloadItem(errors[i]))
	}
	return ep
}

type ExecuteInput struct {
	Command     string `json:"command"`
	Script      string `json:"script"`
	Interpreter string `json:"interpreter"`
	Cwd         string `json:"cwd"`
	IsSudo      bool   `json:"is_sudo"`
	TimeoutSec  int    `json:"timeout_sec"`
	ClientID    string
	IsScript    bool
}

type Meta struct {
	Count int `json:"count"`
}

func NewMeta(count int) *Meta {
	return &Meta{
		Count: count,
	}
}

type Links interface{}
