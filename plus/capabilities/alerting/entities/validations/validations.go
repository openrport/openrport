package validations

type ValidationError struct {
	Prefix string
	Err    error
}

type ErrorList []ValidationError
