package oauth

import (
	"plugin"

	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/logger"
)

const (
	InitOAuthCapabilityEx  = "InitOAuthCapabilityEx"
	GitHubOAuthProvider    = "github"
	MicrosoftOAuthProvider = "microsoft"
	GoogleOAuthProvider    = "google"
	Auth0OAuthProvider     = "auth0"

	DefaultLoginURI       = "/oauth/login"
	DefaultDeviceLoginURI = "/oauth/login/device"
)

// Capability is used by rportd to maintain loaded info about the plugin's
// oauth capability
type Capability struct {
	Provider CapabilityEx

	Config *Config
	Logger *logger.Logger
}

// GetInitFuncName gets the name of the capability init func
func (cap *Capability) GetInitFuncName() (name string) {
	return InitOAuthCapabilityEx
}

// InitProvider invokes the capability init func in the plugin and saves
// the returned capability provider interface. This interface provides
// the functions of the capability.
func (cap *Capability) InitProvider(initFn plugin.Symbol) {
	fn := initFn.(func(cap *Capability) (capProvider CapabilityEx))
	cap.Provider = fn(cap)
}

// GetOAuthCapabilityEx returns the interface to the capability functions
func (cap *Capability) GetOAuthCapabilityEx() (capEx CapabilityEx) {
	return cap.Provider
}

// GetConfigValidator returns a validator interface that can be called to
// validate the capability config
func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return cap.Provider
}
