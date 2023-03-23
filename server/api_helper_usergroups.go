package chserver

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/auditlog"
)

func (al *APIListener) handleListUserGroups(w http.ResponseWriter, req *http.Request) {
	items, err := al.userService.ListGroups()
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(items))
}

func (al *APIListener) handleGetUserGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["group_name"]

	group, err := al.userService.GetGroup(name)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(group))
}

func (al *APIListener) handleUpdateUserGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["group_name"]

	var input users.Group
	err := parseRequestBody(req.Body, &input)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	group, err := al.userService.UpdateGroup(name, input)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationAuthUserGroup, auditlog.ActionUpdate).
		WithHTTPRequest(req).
		WithRequest(input).
		WithResponse(group).
		WithID(name).
		Save()

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(group))
}

func (al *APIListener) handleDeleteUserGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	name := vars["group_name"]

	err := al.userService.DeleteGroup(name)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationAuthUserGroup, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(name).
		Save()

	w.WriteHeader(http.StatusNoContent)
}
