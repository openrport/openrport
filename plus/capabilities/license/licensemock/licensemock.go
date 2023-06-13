package licensemock

import (
	"plugin"

	licensecap "github.com/realvnc-labs/rport/plus/capabilities/license"
	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/logger"
)

const (
	MaxUsers   = 50
	MaxClients = 2000
)

var HasValidLicense bool

type MockCapabilityProvider struct {
}

type Capability struct {
	Provider *MockCapabilityProvider

	Config *licensecap.Config
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

// GetLicenseCapabilityEx returns the mock provider's interface to the capability
// functions
func (cap *Capability) GetLicenseCapabilityEx() (capEx licensecap.CapabilityEx) {
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

// GetMaxUsers returns the max users allowed by the license
func (mp *MockCapabilityProvider) GetMaxUsers() (maxUsers int) {
	if HasValidLicense {
		return MaxUsers
	}
	return 0
}

// GetMaxClients returns the max clients allowed by the license
func (mp *MockCapabilityProvider) GetMaxClients() (maxClients int) {
	if HasValidLicense {
		return MaxClients
	}
	return 0
}

// LicenseInfoAvailable returns true if the license info has been received from the license server
func (mp *MockCapabilityProvider) LicenseInfoAvailable() (avail bool) {
	return HasValidLicense
}

// SetLicenseInfoAvailableNotifier informs the runner of the notify fn to handle when the license info has been received
func (mp *MockCapabilityProvider) SetLicenseInfoAvailableNotifier(notifyFn licensecap.LicenseInfoAvailableNotifier) {
}

// Gets the current trial mode status for rportd
func (mp *MockCapabilityProvider) IsTrialMode() (isTrial bool) {
	return !HasValidLicense
}

// GetLicenseInfo makes the retrieved license info available to rportd
func (mp *MockCapabilityProvider) GetLicenseInfo() (licenseInfo *licensecap.PlusLicenseInfo) {
	if !HasValidLicense {
		return nil
	}

	licenseInfo = &licensecap.PlusLicenseInfo{
		MaxClients: MaxClients,
		MaxUsers:   MaxUsers,
	}
	return licenseInfo
}
