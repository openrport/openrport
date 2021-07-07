package api

import (
	"errors"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

// SuccessPayload represents a uniform format for all successful API responses.
type SuccessPayload struct {
	Data interface{} `json:"data"`
	Meta interface{} `json:"meta,omitempty"`
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

func NewErrorPayloadWithCode(code, title, detail string) ErrorPayload {
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

func NewErrorPayload(err error) ErrorPayload {
	var apiErr errors2.APIError
	var apiErrs errors2.APIErrors
	switch {
	case errors.As(err, &apiErr):
		return newAPIErrorPayload(apiErr)
	case errors.As(err, &apiErrs):
		return newAPIErrorsPayload(apiErrs)
	}

	return NewErrorPayloadWithCode("", err.Error(), "")
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

type ExecuteCommandInput struct {
	Command    string `json:"command"`
	Shell      string `json:"shell"`
	Cwd        string `json:"cwd"`
	IsSudo     bool   `json:"bool"`
	TimeoutSec int    `json:"timeout_sec"`
	ClientID   string
	IsScript   bool
}
