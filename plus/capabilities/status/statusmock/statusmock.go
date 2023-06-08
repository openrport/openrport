package statusmock

import (
	"plugin"

	"github.com/realvnc-labs/rport/plus/capabilities/status"
	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/logger"
)

type MockCapabilityProvider struct {
}

type Capability struct {
	Provider *MockCapabilityProvider

	Config *status.Config
	Logger *logger.Logger
}

// GetInitFuncName return the empty string as the mock capability doesn't use the plugin
func (cap *Capability) GetInitFuncName() (name string) {
	return ""
}

// InitProvider sets the capability provider to the local mock implementation
func (cap *Capability) InitProvider(initFn plugin.Symbol) {
	if cap.Provider == nil {
		cap.Provider = &MockCapabilityProvider{}
	}
}

// GetStatusCapabilityEx returns the mock provider's interface to the capability
// functions
func (cap *Capability) GetStatusCapabilityEx() (capEx status.CapabilityEx) {
	return cap.Provider
}

// GetConfigValidator returns a validator interface that can be called to
// validate the capability config
func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return cap.Provider
}

// ValidateConfig does nothing for the mock implementation
func (mp *MockCapabilityProvider) ValidateConfig() (err error) {
	return nil
}

// GetStatusInfo returns mock status info
func (mp *MockCapabilityProvider) GetStatusInfo() (info *status.PlusStatusInfo) {
	info = &status.PlusStatusInfo{}
	return info
}
