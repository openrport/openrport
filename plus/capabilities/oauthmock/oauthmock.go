package oauthmock

import (
	"errors"
	"net/http"
	"plugin"
	"time"

	"github.com/realvnc-labs/rport/plus/capabilities/oauth"
	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/logger"
)

type MockCapabilityProvider struct {
	PerformAuthCodeExchangeRequest    *http.Request
	GetUserToken                      string
	ShouldFailGetLoginInfo            bool
	ShouldFailGetAccessTokenForDevice bool
	Username                          string
}

type Capability struct {
	Provider *MockCapabilityProvider

	Config *oauth.Config
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

// GetLoginInfo returns mock login info
func (mp *MockCapabilityProvider) GetLoginInfo() (loginInfo *oauth.LoginInfo, err error) {
	if mp.ShouldFailGetLoginInfo {
		return nil, errors.New("got an error")
	}
	loginInfo = &oauth.LoginInfo{
		LoginMsg:     "mock login msg",
		AuthorizeURL: "mock authorize url",
		LoginURI:     "/mock_login_uri",
		State:        "123456",
		Expiry:       time.Time{},
	}
	return loginInfo, nil
}

// PerformAuthCodeExchange saves the received request and returns a mock token
func (mp *MockCapabilityProvider) PerformAuthCodeExchange(r *http.Request) (token string, username string, err error) {
	mp.PerformAuthCodeExchangeRequest = r
	return "mock token", "", nil
}

// GetPermittedUser saves the token received and returns the configured mock username
func (mp *MockCapabilityProvider) GetPermittedUser(r *http.Request, token string) (username string, err error) {
	mp.GetUserToken = token
	if mp.Username == "" {
		username = "mock-user"
	} else {
		username = mp.Username
	}
	return username, nil
}

func (mp *MockCapabilityProvider) GetLoginInfoForDevice(r *http.Request) (loginInfo *oauth.DeviceLoginInfo, err error) {
	if mp.ShouldFailGetLoginInfo {
		return nil, errors.New("got an error")
	}

	authInfo := &oauth.DeviceAuthInfo{
		UserCode:        "mock-user-code",
		DeviceCode:      "mock-device-code",
		VerificationURI: "mock-verification-uri",
		ExpiresIn:       333,
		Interval:        4,
		Message:         "mock-message",
	}

	loginInfo = &oauth.DeviceLoginInfo{
		LoginURI:       "/mock_device_login_uri",
		DeviceAuthInfo: authInfo,
	}

	return loginInfo, nil
}

func (mp *MockCapabilityProvider) GetAccessTokenForDevice(r *http.Request) (token string, username string, errInfo *oauth.DeviceAuthStatusErrorInfo, err error) {
	if mp.ShouldFailGetAccessTokenForDevice {
		errInfo := &oauth.DeviceAuthStatusErrorInfo{
			StatusCode:   http.StatusForbidden,
			ErrorCode:    "got an error",
			ErrorMessage: "error message",
			ErrorURI:     "https://error-info-here.com",
		}
		return "", "", errInfo, errors.New("got an error")
	}

	return "mock-token", "mock-username", nil, nil
}

func (mp *MockCapabilityProvider) GetPermittedUserForDevice(t *http.Request, token string) (username string, err error) {
	return "testuser", nil
}
