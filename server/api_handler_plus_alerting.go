package chserver

import (
	"net/http"

	rportplus "github.com/realvnc-labs/rport/plus"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rules"
)

func (al *APIListener) handleSaveRuleSet(w http.ResponseWriter, r *http.Request) {
	plus := al.Server.plusManager
	if plus == nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, rportplus.ErrPlusNotAvailable)
		return
	}

	capEx := plus.GetAlertingCapabilityEx()
	if capEx == nil {
		al.jsonErrorResponse(w, http.StatusForbidden, rportplus.ErrCapabilityNotAvailable(rportplus.PlusAlertingCapability))
		return
	}

	as := capEx.GetService()

	rs := &rules.RuleSet{}

	err := parseRequestBody(r.Body, &rs)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.Debugf("saving ruleset = %v", rs)

	err = as.SaveRuleSet(rs)
	if err != nil {
		// TODO: (rs): test if this needs to wrap the error with some description/context?
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
	}

	err = as.LoadLatestRuleSet()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
	}
}

func (al *APIListener) handleGetActiveRuleActionStates(w http.ResponseWriter, r *http.Request) {
	plus := al.Server.plusManager
	if plus == nil {
		al.jsonErrorResponse(w, http.StatusUnauthorized, rportplus.ErrPlusNotAvailable)
		return
	}

	capEx := plus.GetAlertingCapabilityEx()
	if capEx == nil {
		al.jsonErrorResponse(w, http.StatusForbidden, rportplus.ErrCapabilityNotAvailable(rportplus.PlusAlertingCapability))
		return
	}

	as := capEx.GetService()

	// TODO: (rs): what should this limit be?
	states, err := as.GetLatestRuleActionStates(1000)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
	}
	al.writeJSONResponse(w, http.StatusOK, states)
}
