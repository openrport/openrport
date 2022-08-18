package chserver

import (
	"net/http"

	rportplus "github.com/cloudradar-monitoring/rport/rport-plus"
	"github.com/cloudradar-monitoring/rport/server/api"
)

// handleOAuthAuthorizationCode takes a request containing the OAuth authorization code parameters
// and sends it to the rport-plus plugin to be exchanged for valid github access token and then username.
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

	token, err := capEx.PerformAuthCodeExchange(r)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, err)
		return
	}

	username, err := capEx.GetValidUser(token)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, err)
		return
	}

	// pass the username to the existing login logic to create the user (if required) and
	// get an rport-plus bearer token.
	al.handleLogin(username, "", true /* skipPasswordValidation */, w, r)
}

// handlePlusVersion makes a request to the plugin for it's version info
func (al *APIListener) handlePlusVersion(w http.ResponseWriter, r *http.Request) {
	plus := al.Server.plusManager
	if plus == nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, rportplus.ErrPlusNotAvailable)
		return
	}

	capEx := plus.GetVersionCapabilityEx()
	if capEx == nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, rportplus.ErrCapabilityNotAvailable(rportplus.PlusVersionCapability))
		return
	}

	versionInfo := capEx.GetVersionInfo()

	response := api.NewSuccessPayload(versionInfo)
	al.writeJSONResponse(w, http.StatusOK, response)
}

// TODO: if not going to be used then this can be deleted
func (al *APIListener) handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
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

	capEx.HandleLogin(w, r)
}
