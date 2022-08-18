package validator

// Validator provides a narrow interface for capability validation
type Validator interface {
	ValidateConfig() (err error)
}
