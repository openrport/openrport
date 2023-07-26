package validations

type ValidationError struct {
	Prefix string `json:"prefix"`
	Err    error  `json:"err"`
}

type ErrorList []ValidationError
