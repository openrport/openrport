package chserver

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/routes"
	"github.com/cloudradar-monitoring/rport/share/query"
	"github.com/cloudradar-monitoring/rport/share/types"
)

func (al *APIListener) handlePostClientGroups(w http.ResponseWriter, req *http.Request) {
	var group cgroups.ClientGroup
	err := parseRequestBody(req.Body, &group)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	if err := validateInputClientGroup(group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "Invalid client group.", err)
		return
	}

	if err := al.clientGroupProvider.Create(req.Context(), &group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to persist a new client group.", err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationClientGroup, auditlog.ActionCreate).
		WithHTTPRequest(req).
		WithRequest(group).
		WithID(group.ID).
		Save()

	w.WriteHeader(http.StatusCreated)
	al.Debugf("Client Group [id=%q] created.", group.ID)
}

func (al *APIListener) handlePutClientGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars[routes.ParamGroupID]
	if id == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routes.ParamGroupID))
		return
	}

	var group cgroups.ClientGroup
	err := parseRequestBody(req.Body, &group)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	if id != group.ID {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("%q route param doesn't not match group ID from request body.", routes.ParamGroupID))
		return
	}

	if err := validateInputClientGroup(group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "Invalid client group.", err)
		return
	}

	if err := al.clientGroupProvider.Update(req.Context(), &group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to persist client group.", err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationClientGroup, auditlog.ActionUpdate).
		WithHTTPRequest(req).
		WithRequest(group).
		WithID(id).
		Save()

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("Client Group [id=%q] updated.", group.ID)
}

const groupIDMaxLength = 30
const validGroupIDChars = "A-Za-z0-9_-*"

var invalidGroupIDRegexp = regexp.MustCompile(`[^\*A-Za-z0-9_-]`)

func validateInputClientGroup(group cgroups.ClientGroup) error {
	if strings.TrimSpace(group.ID) == "" {
		return errors.New("group ID cannot be empty")
	}
	if len(group.ID) > groupIDMaxLength {
		return fmt.Errorf("invalid group ID: max length %d, got %d", groupIDMaxLength, len(group.ID))
	}
	if invalidGroupIDRegexp.MatchString(group.ID) {
		return fmt.Errorf("invalid group ID %q: can contain only %q", group.ID, validGroupIDChars)
	}
	return nil
}

func (al *APIListener) handleGetClientGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars[routes.ParamGroupID]
	if id == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routes.ParamGroupID))
		return
	}

	options := query.GetRetrieveOptions(req)
	err := query.ValidateRetrieveOptions(options, cgroups.OptionsSupportedFields)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	requestedFields := query.RequestedFields(options.Fields, cgroups.OptionsResource)

	group, err := al.clientGroupProvider.Get(req.Context(), id)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to find client group[id=%q].", id), err)
		return
	}
	if group == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Client Group[id=%q] not found.", id))
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.clientService.PopulateGroupsWithUserClients([]*cgroups.ClientGroup{group}, curUser)

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(convertToClientGroupPayload(group, requestedFields)))
}

func (al *APIListener) handleGetClientGroups(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	options := query.NewOptions(req, cgroups.OptionsListDefaultSort, nil /* filtersDefault */, cgroups.OptionsListDefaultFields)

	err := query.ValidateListOptions(options, cgroups.OptionsSupportedFiltersAndSorts, cgroups.OptionsSupportedFiltersAndSorts, cgroups.OptionsSupportedFields, &query.PaginationConfig{
		MaxLimit:     500,
		DefaultLimit: 50,
	})
	if err != nil {
		al.jsonError(w, err)
		return
	}

	// pagination and fields are not done in db, because of filterEmptyGroups
	pagination := options.Pagination
	options.Pagination = nil
	requestedFields := query.RequestedFields(options.Fields, cgroups.OptionsResource)
	options.Fields = nil

	groups, err := al.clientGroupProvider.List(ctx, options)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to get client groups.", err)
		return
	}

	curUser, err := al.getUserModelForAuth(ctx)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.clientService.PopulateGroupsWithUserClients(groups, curUser)

	// for non-admins filter out groups with no clients
	if !curUser.IsAdmin() {
		groups = filterEmptyGroups(groups)
	}

	totalCount := len(groups)
	start, end := pagination.GetStartEnd(totalCount)
	limited := groups[start:end]

	al.writeJSONResponse(w, http.StatusOK, &api.SuccessPayload{
		Data: convertToClientGroupsPayload(limited, requestedFields),
		Meta: api.NewMeta(len(groups)),
	})
}

func filterEmptyGroups(groups []*cgroups.ClientGroup) []*cgroups.ClientGroup {
	var nonEmptyGroups []*cgroups.ClientGroup
	for _, group := range groups {
		if len(group.ClientIDs) > 0 {
			nonEmptyGroups = append(nonEmptyGroups, group)
		}
	}
	return nonEmptyGroups
}

func (al *APIListener) handleDeleteClientGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars[routes.ParamGroupID]
	if id == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routes.ParamGroupID))
		return
	}

	err := al.clientGroupProvider.Delete(req.Context(), id)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete client group[id=%q].", id), err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationClientGroup, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(id).
		Save()

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("Client Group [id=%q] deleted.", id)
}

type ClientGroupPayload struct {
	ID                *string               `json:"id,omitempty"`
	Description       *string               `json:"description,omitempty"`
	Params            *cgroups.ClientParams `json:"params,omitempty" db:"params"`
	AllowedUserGroups *types.StringSlice    `json:"allowed_user_groups,omitempty"`
	ClientIDs         *[]string             `json:"client_ids,omitempty" db:"-"`
}

func convertToClientGroupsPayload(clientGroups []*cgroups.ClientGroup, requestedFields map[string]bool) []ClientGroupPayload {
	r := make([]ClientGroupPayload, 0, len(clientGroups))
	for _, cur := range clientGroups {
		r = append(r, convertToClientGroupPayload(cur, requestedFields))
	}
	return r
}

func convertToClientGroupPayload(clientGroup *cgroups.ClientGroup, requestedFields map[string]bool) ClientGroupPayload {
	p := ClientGroupPayload{}
	for field := range cgroups.OptionsSupportedFields[cgroups.OptionsResource] {
		if len(requestedFields) > 0 && !requestedFields[field] {
			continue
		}
		switch field {
		case "id":
			p.ID = &clientGroup.ID
		case "description":
			p.Description = &clientGroup.Description
		case "params":
			p.Params = clientGroup.Params
		case "allowed_user_groups":
			p.AllowedUserGroups = &clientGroup.AllowedUserGroups
		case "client_ids":
			p.ClientIDs = &clientGroup.ClientIDs
		}
	}
	return p
}
