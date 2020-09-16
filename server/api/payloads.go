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
