package oauthmock

import (
	"net/http"
	"plugin"

	"github.com/cloudradar-monitoring/rport/rport-plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/rport-plus/validator"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

type MockCapabilityProvider struct {
	PerformAuthCodeExchangeRequest *http.Request
	GetUserToken                   string

	Username string
}

// Capability is used by rportd to maintain loaded info about the plugin's
// oauth capability
type Capability struct {
	Provider *MockCapabilityProvider

	Config *oauth.Config
	Logger *logger.Logger
}

// GetInitFuncName return the empty string as the mock capability does use the plugin
func (cap *Capability) GetInitFuncName() (name string) {
	return ""
}

// SetProvider sets the capability provider to the local mock implementation
func (cap *Capability) SetProvider(initFn plugin.Symbol) {
	if cap.Provider == nil {
		cap.Provider = &MockCapabilityProvider{}
	}
}

// GetOAuthCapabilityEx returns the mock provider's interface to the capability
// functions
func (cap *Capability) GetOAuthCapabilityEx() (capEx oauth.CapabilityEx) {
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

// GetOAuthLoginInfo returns dummy login info
func (mp *MockCapabilityProvider) GetOAuthLoginInfo() (loginMsg string, loginURL string, state string, err error) {
	return "dummy login msg", "dummy login url", "dummy state", nil
}

// HandleLogin...
func (mp *MockCapabilityProvider) HandleLogin(w http.ResponseWriter, r *http.Request) {}

// PerformAuthCodeExchange...
func (mp *MockCapabilityProvider) PerformAuthCodeExchange(r *http.Request) (token string, err error) {
	mp.PerformAuthCodeExchangeRequest = r
	return "mock token", nil
}

// GetValidUser...
func (mp *MockCapabilityProvider) GetValidUser(token string) (username string, err error) {
	mp.GetUserToken = token
	if mp.Username == "" {
		username = "mock-user"
	} else {
		username = mp.Username
	}
	return username, nil
}
