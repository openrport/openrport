package chserver

import (
	"net/http"

	rportplus "github.com/cloudradar-monitoring/rport/plus"
	"github.com/cloudradar-monitoring/rport/server/api"
)

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
		al.jsonErrorResponse(w, http.StatusForbidden, rportplus.ErrCapabilityNotAvailable(rportplus.PlusOAuthCapability))
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
		al.jsonErrorResponse(w, http.StatusForbidden, rportplus.ErrPlusNotAvailable)
		return
	}

	capEx := plus.GetStatusCapabilityEx()
	if capEx == nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, rportplus.ErrCapabilityNotAvailable(rportplus.PlusStatusCapability))
		return
	}

	statusInfo := capEx.GetStatusInfo()

	response := api.NewSuccessPayload(statusInfo)
	al.writeJSONResponse(w, http.StatusOK, response)
}
