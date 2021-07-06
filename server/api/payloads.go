package api

import (
	"strconv"

	"github.com/cloudradar-monitoring/rport/server/api/errors"
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
	return NewErrorPayloadWithCode("", "", err.Error())
}

func NewAPIErrorsPayloadWithCode(errors []errors.APIError) ErrorPayload {
	ep := ErrorPayload{
		Errors: make([]ErrorPayloadItem, 0, len(errors)),
	}
	for i := range errors {
		ep.Errors = append(ep.Errors, ErrorPayloadItem{
			Code:   strconv.Itoa(errors[i].Code),
			Title:  errors[i].Error(),
			Detail: "",
		})
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
