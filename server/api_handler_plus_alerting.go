package chserver

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/validations"
	"github.com/realvnc-labs/rport/server/alerts"

	rportplus "github.com/realvnc-labs/rport/plus"
	alertingcap "github.com/realvnc-labs/rport/plus/capabilities/alerting"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rules"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/templates"
	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/routes"
	"github.com/realvnc-labs/rport/share/query"
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

func (al *APIListener) handleGetRuleSet(w http.ResponseWriter, _ *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	rs, err := as.LoadRuleSet(rules.DefaultRuleSetID)
	if err != nil && !errors.Is(err, alertingcap.ErrEntityNotFound) {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if rs == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "no ruleset available")
		return
	}

	al.Debugf("loaded ruleset")
	rs.RuleSetID = ""

	response := api.NewSuccessPayload(rs)

	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleDeleteRuleSet(w http.ResponseWriter, _ *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	err = as.DeleteRuleSet(rules.DefaultRuleSetID)
	if err != nil {
		if errors.Is(err, alertingcap.ErrEntityNotFound) {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "no ruleset found")
			return
		}
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.Debugf("deleted ruleset = %s", rules.DefaultRuleSetID)
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

	if r.Method == "PUT" {
		if rs.RuleSetID != "" {
			al.jsonErrorResponse(w, http.StatusBadRequest, errors.New("when saving rules via PUT, the id in request body must be omitted or empty"))
			return
		}
	}

	rs.RuleSetID = rules.DefaultRuleSetID

	errs, err := as.SaveRuleSet(rs)
	if err != nil {
		if errs != nil {
			errPayload := makeValidationErrorPayload(errs)
			al.writeJSONResponse(w, http.StatusBadRequest, errPayload)
			return
		}
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	err = as.LoadDefaultRuleSet()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.Debugf("saved ruleset = %v", rs)
}

func makeValidationErrorPayload(errs validations.ErrorList) *api.ErrorPayload {
	validationErrs := []api.ErrorPayloadItem{}
	for _, validationErr := range errs {
		vErr := api.ErrorPayloadItem{
			Code:   "",
			Title:  "error during rule set validation",
			Detail: fmt.Sprintf("%s: %s", validationErr.Prefix, validationErr.Err.Error()),
		}
		validationErrs = append(validationErrs, vErr)
	}
	errPayload := &api.ErrorPayload{
		Errors: validationErrs,
	}
	return errPayload
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

	if r.Method == "POST" {
		_, err := as.GetTemplate(template.ID)
		if err == nil {
			al.jsonErrorResponse(w, http.StatusConflict, errors.New("template exists"))
			return
		}

		if !errors.Is(err, alertingcap.ErrEntityNotFound) {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
	}

	if r.Method == "PUT" {
		if template.ID != "" {
			al.jsonErrorResponse(w, http.StatusBadRequest, errors.New("when saving existing templates, the template id in request body must be omitted or empty"))
			return
		}

		vars := mux.Vars(r)
		tid := vars[routes.ParamTemplateID]

		template.ID = templates.TemplateID(tid)
	}

	errs, err := as.SaveTemplate(template)
	if err != nil {
		if errs != nil {
			errPayload := makeValidationErrorPayload(errs)
			al.writeJSONResponse(w, http.StatusBadRequest, errPayload)
			return
		}
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.Debugf("saved template = %v", template)
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
		if errors.Is(err, alertingcap.ErrEntityNotFound) {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("template with id %q not found", tid))
			return
		}
		if errors.Is(err, templates.ErrTemplateInUse) {
			al.jsonErrorResponseWithTitle(w, http.StatusForbidden, err.Error())
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
	if err != nil && !errors.Is(err, alertingcap.ErrEntityNotFound) {
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

func (al *APIListener) handleGetAllTemplates(w http.ResponseWriter, _ *http.Request) {
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

func (al *APIListener) handleGetProblem(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	vars := mux.Vars(r)
	problemID := vars[routes.ParamProblemID]

	problem, err := as.GetProblem(rules.ProblemID(problemID))
	if err != nil && !errors.Is(err, alertingcap.ErrEntityNotFound) {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if problem == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("problem with id %s not found", problemID))
		return
	}

	al.Debugf("loaded problem = %v", problem)

	response := api.NewSuccessPayload(problem)

	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleUpdateProblem(w http.ResponseWriter, r *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	problemUpdateRequest := &rules.ProblemUpdateRequest{}

	err = parseRequestBody(r.Body, &problemUpdateRequest)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	vars := mux.Vars(r)
	pid := vars[routes.ParamProblemID]
	problemID := rules.ProblemID(pid)

	if problemUpdateRequest.Active {
		err = as.SetProblemActive(problemID)
		if err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
	} else {
		err = as.SetProblemResolved(problemID, time.Now().UTC())
		if err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
	}
	al.Debugf("updated problem = %v", problemUpdateRequest)
}

func (al *APIListener) handleGetLatestProblems(w http.ResponseWriter, req *http.Request) {
	as, status, err := al.getAlertingService()
	if err != nil {
		al.jsonErrorResponse(w, status, err)
	}

	options := query.NewOptions(req, nil, nil, alerts.SupportedProblemsListFields)
	errs := query.ValidateListOptions(options,
		alerts.SupportedProblemsSorts,
		alerts.SupportedProblemsFilters,
		alerts.SupportedProblemsFields,
		&query.PaginationConfig{
			MaxLimit:     500,
			DefaultLimit: 50,
		})

	if errs != nil {
		al.jsonError(w, errs)
		return
	}

	sortFunc, desc, err := alerts.SortProblemsFunc(options.Sorts)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	problems, err := as.GetLatestProblems(alertingcap.NoLimit)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	// TODO: (rs):  these filters are applied against ALL existing problems loaded into RAM. Need to decide
	// the long term approach and whether to implement via queries against the DB directly?

	matchingProblems := make([]*rules.Problem, 0, 128)

	for _, problem := range problems {
		matches, err := query.MatchesFilters(problem, options.Filters)

		if err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if matches {
			matchingProblems = append(matchingProblems, problem)
		}
	}

	sortFunc(matchingProblems, desc)

	totalCount := len(matchingProblems)
	start, end := options.Pagination.GetStartEnd(totalCount)
	pagedProblems := matchingProblems[start:end]

	al.Debugf("total problems = %d", len(pagedProblems))

	response := &api.SuccessPayload{
		Data: pagedProblems,
		Meta: api.NewMeta(totalCount),
	}

	al.writeJSONResponse(w, http.StatusOK, response)
}
