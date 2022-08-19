package validator

// Validator provides a narrow interface for capability validation.
// In a separate package to avoid circular dependencies.
type Validator interface {
	ValidateConfig() (err error)
}
