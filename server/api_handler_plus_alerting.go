package chserver

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	rportplus "github.com/realvnc-labs/rport/plus"
	alertingcap "github.com/realvnc-labs/rport/plus/capabilities/alerting"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rules"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/templates"
	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/routes"
)

func (al *APIListener) getAlertingService() (as alertingcap.Service, statusCode int, err error) {
	plusManager := al.Server.plusManager
	if plusManager == nil {
		return nil, http.StatusUnauthorized, rportplus.ErrPlusNotAvailable
	}

	capEx := plusManager.GetAlertingCapabilityEx()
	if capEx == nil {
		return nil, http.StatusForbidden, rportplus.ErrCapabilityNotAvailable(rportplus.PlusAlertingCapability)
	}

	as = capEx.GetService()

	return as, 0, nil
}

func (al *APIListener) handleGetRuleSet(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	vars := mux.Vars(r)
	ruleSetID := vars[routes.ParamRuleSetID]

	rs, err := as.LoadRuleSet(rules.RuleSetID(ruleSetID))
	if err != nil && !errors.Is(err, rules.ErrRuleSetNotFound) {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if rs == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("ruleset with id %q not found", ruleSetID))
		return
	}

	al.Debugf("loaded ruleset = %v", rs)

	response := api.NewSuccessPayload(rs)

	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleDeleteRuleSet(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	vars := mux.Vars(r)
	ruleSetID := vars[routes.ParamRuleSetID]

	err = as.DeleteRuleSet(rules.RuleSetID(ruleSetID))
	if err != nil {
		if errors.Is(err, rules.ErrRuleSetNotFound) {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("ruleset with id %q not found", ruleSetID))
			return
		}
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.Debugf("deleted ruleset = %s", ruleSetID)
}

func (al *APIListener) handleSaveRuleSet(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	rs := &rules.RuleSet{}

	err = parseRequestBody(r.Body, &rs)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.Debugf("saving ruleset = %v", rs)

	err = as.SaveRuleSet(rs)
	if err != nil {
		// TODO: (rs): test if this needs to wrap the error with some description/context?
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	err = as.LoadLatestRuleSet()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
}

func (al *APIListener) handleSaveTemplate(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	template := &templates.Template{}

	err = parseRequestBody(r.Body, &template)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.Debugf("saving template = %v", template)

	err = as.SaveTemplate(template)
	if err != nil {
		// TODO: (rs): test if this needs to wrap the error with some description/context?
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
}

func (al *APIListener) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	vars := mux.Vars(r)
	tid := vars[routes.ParamTemplateID]

	err = as.DeleteTemplate(templates.TemplateID(tid))
	if err != nil {
		if errors.Is(err, templates.ErrTemplateNotFound) {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("template with id %q not found", tid))
			return
		}
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.Debugf("deleted template = %s", tid)
}

func (al *APIListener) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	vars := mux.Vars(r)
	tid := vars[routes.ParamTemplateID]

	template, err := as.GetTemplate(templates.TemplateID(tid))
	if err != nil && !errors.Is(err, templates.ErrTemplateNotFound) {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if template == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("template with id %q not found", tid))
		return
	}

	al.Debugf("loaded template = %v", template)

	response := api.NewSuccessPayload(template)

	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleGetAllTemplates(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	templateList, err := as.GetAllTemplates()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.Debugf("loaded templates = %v", templateList)

	response := api.NewSuccessPayload(templateList)

	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleGetActiveRuleActionStates(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	// TODO: (rs): what should this limit be?
	states, err := as.GetLatestRuleActionStates(1000)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := api.NewSuccessPayload(states)

	al.writeJSONResponse(w, http.StatusOK, response)
}
