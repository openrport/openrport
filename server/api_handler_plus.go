package chserver

import (
	"net/http"

	rportplus "github.com/cloudradar-monitoring/rport/rport-plus"
	"github.com/cloudradar-monitoring/rport/server/api"
)

type LoginInfoResponse struct {
	Errors []LoginInfoPayload `json:"errors"`
}

type LoginInfoPayload struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`

	LoginURL    string `json:"login_url"`
	ExchangeURI string `json:"exchange_uri"`
}

func (al *APIListener) jsonLoginInfoResponse(w http.ResponseWriter, loginMsg string, loginURL string, exchangeURI string) {
	loginInfo := LoginInfoResponse{
		Errors: []LoginInfoPayload{{
			Code:     "",
			Title:    "Unauthorized",
			Detail:   loginMsg,
			LoginURL: loginURL,
			// TODO: is the rport server domain available for using here?
			ExchangeURI: exchangeURI,
		},
		},
	}
	al.writeJSONResponse(w, http.StatusUnauthorized, loginInfo)
}

// note that the plugin not actually involved here
func (al *APIListener) handleOAuthStatusRequest(w http.ResponseWriter, r *http.Request) {
	cfg := *al.Server.config.OAuthConfig
	cfg.ClientSecret = ""
	response := api.NewSuccessPayload(cfg)
	al.writeJSONResponse(w, http.StatusOK, response)
}

// handleOAuthAuthorizationCode takes a request containing the OAuth authorization code parameters
// and sends it to the rport-plus plugin to be exchanged for valid access token and then username.
// The username can then be used to create a user (if required) and obtain a rport bearer token
func (al *APIListener) handleOAuthAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	plus := al.Server.plusManager
	if plus == nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, rportplus.ErrPlusNotAvailable)
		return
	}

	capEx := plus.GetOAuthCapabilityEx()
	if capEx == nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, rportplus.ErrCapabilityNotAvailable(rportplus.PlusOAuthCapability))
		return
	}

	// if IDToken in use then possible that a permitted username
	// will be returned after the auth code exchange. if the user
	// isn't permitted then an error will be returned.
	token, username, err := capEx.PerformAuthCodeExchange(r)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, err)
		return
	}

	// if no previous err and an empty username then attempt to get
	// a permitted username from the oauth/identity provider
	if username == "" {
		username, err = capEx.GetPermittedUser(r, token)
		if err != nil {
			al.jsonErrorResponse(w, http.StatusUnauthorized, err)
			return
		}
	}

	// pass the username to the existing login logic to create the user (if required) and
	// get an rport-plus bearer token.
	al.handleLogin(username, "", true /* skipPasswordValidation */, w, r)
}

// handlePlusStatus makes a request to the plugin for it's status/version info
func (al *APIListener) handlePlusStatus(w http.ResponseWriter, r *http.Request) {
	plus := al.Server.plusManager
	if plus == nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, rportplus.ErrPlusNotAvailable)
		return
	}

	capEx := plus.GetStatusCapabilityEx()
	if capEx == nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, rportplus.ErrCapabilityNotAvailable(rportplus.PlusStatusCapability))
		return
	}

	statusInfo := capEx.GetStatusInfo()

	response := api.NewSuccessPayload(statusInfo)
	al.writeJSONResponse(w, http.StatusOK, response)
}
