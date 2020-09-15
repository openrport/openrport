package api

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
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

func NewErrorPayloadWithCode(err error, code string) ErrorPayload {
	return ErrorPayload{
		Errors: []ErrorPayloadItem{
			{
				Detail: err.Error(),
				Code:   code,
			},
		},
	}
}

func NewErrorPayload(err error) ErrorPayload {
	return NewErrorPayloadWithCode(err, "")
}
