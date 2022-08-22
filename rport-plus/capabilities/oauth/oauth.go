package oauth

import (
	"errors"
	"net/http"
	"plugin"

	"github.com/cloudradar-monitoring/rport/rport-plus/validator"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

var (
	ErrMissingConfig       = errors.New("no config to validate")
	ErrMissingProvider     = errors.New("missing provider")
	ErrMissingAuthorizeURL = errors.New("missing authorize_url")
	ErrMissingTokenURL     = errors.New("missing token_url")
	ErrMissingRedirectURI  = errors.New("missing redirect_uri")
	ErrMissingClientID     = errors.New("missing client_id")
	ErrMissingClientSecret = errors.New("missing client_secret")
)

// CapabilityEx represents the functional interface provided by the OAuth plugin
type CapabilityEx interface {
	ValidateConfig() (err error)
	GetOAuthLoginInfo() (loginMsg string, loginURL string, state string, err error)
	HandleLogin(w http.ResponseWriter, r *http.Request)
	PerformAuthCodeExchange(r *http.Request) (token string, err error)
	GetValidUser(token string) (username string, err error)
}

// Config is the plugin OAuth config, as loaded from the rportd config file
type Config struct {
	Provider             string `mapstructure:"provider"`
	AuthorizeURL         string `mapstructure:"authorize_url"`
	TokenURL             string `mapstructure:"token_url"`
	RedirectURI          string `mapstructure:"redirect_uri"`
	ClientID             string `mapstructure:"client_id"`
	ClientSecret         string `mapstructure:"client_secret"`
	UseAuthFile          bool   `mapstructure:"use_api_auth_file"`
	RequiredOrganization string `mapstructure:"required_organization"`
	CreateMissingUsers   bool   `mapstructure:"create_missing_users"`
	ProvideOAuthLogin    bool   `mapstructure:"provide_oauth_login"`
	UsePKCE              bool   `mapstructure:"use_pkce"`
}

const (
	InitOAuthCapabilityEx = "InitOAuthCapabilityEx"
	GitHubOAuthProvider   = "github"
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

// SetProvider invokes the capability init func in the plugin and saves
// the returned capability provider interface. This interface provides
// the functions of the capability.
func (cap *Capability) SetProvider(initFn plugin.Symbol) {
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
