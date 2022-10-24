package oauth

import (
	"errors"
	"net/http"
	"regexp"
	"time"
)

var (
	ErrMissingConfig       = errors.New("no config to validate")
	ErrMissingProvider     = errors.New("missing provider")
	ErrMissingAuthorizeURL = errors.New("missing authorize_url")
	ErrMissingTokenURL     = errors.New("missing token_url")
	ErrMissingRedirectURI  = errors.New("missing redirect_uri")
	ErrMissingClientID     = errors.New("missing client_id")
	ErrMissingClientSecret = errors.New("missing client_secret")

	ErrInvalidAuthorizeURL       = errors.New("invalid authorize_url")
	ErrInvalidDeviceAuthorizeURL = errors.New("invalid device_authorize_url")
	ErrInvalidTokenURL           = errors.New("invalid token_url")
	ErrInvalidRedirectURI        = errors.New("invalid redirect_uri")

	ErrNeedPermittedUserControl  = errors.New("permitted_user_list OR permitted_user_match must be specified")
	ErrInvalidPermittedUserMatch = errors.New("invalid permitted_user_match")

	ErrProviderNotSupportedMsg = "not currently supported"
)

// LoginInfo contains the info returned when getting auth settings
// for the web app style flow
type LoginInfo struct {
	LoginMsg     string    `json:"message"`
	AuthorizeURL string    `json:"authorize_url"`
	LoginURI     string    `json:"login_uri"`
	State        string    `json:"state"`
	Expiry       time.Time `json:"expiry"`
}

// DeviceAuthInfo contains the info returned when getting auth settings
// for the device style flow
type DeviceAuthInfo struct {
	UserCode        string `json:"user_code"`
	DeviceCode      string `json:"device_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// GoogleDeviceAuthInfo contains the user auth info google returns for
// the OAuth device flow. Google doesn't follow the standard and
// returns a verification_url rather than verification_uri. The Plus
// plugin maps this to a verification_uri and returns consistent
// DeviceAuthInfo.
type GoogleDeviceAuthInfo struct {
	UserCode        string `json:"user_code"`
	DeviceCode      string `json:"device_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// DeviceLoginInfo represents the login info required for a user
// to login using the device style flow.
type DeviceLoginInfo struct {
	LoginURI string `json:"login_uri"`

	DeviceAuthInfo *DeviceAuthInfo `json:"auth_info"`
}

// DeviceAuthStatusErrorInfo contains any error info returned
// when getting the login info (aka checking the auth status) for the device
// style flow.
type DeviceAuthStatusErrorInfo struct {
	StatusCode   int    `json:"status_code"`
	ErrorCode    string `json:"error"`
	ErrorMessage string `json:"error_description"`
	ErrorURI     string `json:"error_uri"`
}

// CapabilityEx represents the functional interface provided by the OAuth
// capability
type CapabilityEx interface {
	ValidateConfig() (err error)

	GetLoginInfo() (loginInfo *LoginInfo, err error)
	PerformAuthCodeExchange(r *http.Request) (token string, username string, err error)
	GetPermittedUser(r *http.Request, accessToken string) (username string, err error)

	GetLoginInfoForDevice(r *http.Request) (loginInfo *DeviceLoginInfo, err error)
	GetAccessTokenForDevice(r *http.Request) (token string, username string, errInfo *DeviceAuthStatusErrorInfo, err error)
	GetPermittedUserForDevice(r *http.Request, accessToken string) (username string, err error)
}

// Config is the OAuth capability config, as loaded from the rportd config file
type Config struct {
	Provider             string `mapstructure:"provider"`
	BaseAuthorizeURL     string `mapstructure:"authorize_url"`
	TokenURL             string `mapstructure:"token_url"`
	RedirectURI          string `mapstructure:"redirect_uri"`
	ClientID             string `mapstructure:"client_id"`
	ClientSecret         string `mapstructure:"client_secret"`
	RequiredOrganization string `mapstructure:"required_organization"`
	RequiredGroupID      string `mapstructure:"required_group_id"`
	PermittedUserList    bool   `mapstructure:"permitted_user_list"`
	PermittedUserMatch   string `mapstructure:"permitted_user_match"`

	// must be set when the device/cli flow is required.
	// e.g. when using RPort CLI
	BaseDeviceAuthorizeURL string `mapstructure:"device_authorize_url"`

	// these two fields only required when using Google's device flow
	DeviceClientID     string `mapstructure:"device_client_id"`
	DeviceClientSecret string `mapstructure:"device_client_secret"`

	// currently only used by the Auth0 provider
	JWKSURL       string `mapstructure:"jwks_url"`
	RoleClaim     string `mapstructure:"role_claim"`
	RequiredRole  string `mapstructure:"required_role"`
	UsernameClaim string `mapstructure:"username_claim"`

	CompiledPermittedUserMatch *regexp.Regexp
}
