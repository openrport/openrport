package chserver

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/clientsauth"
	"github.com/cloudradar-monitoring/rport/server/routes"
	"github.com/cloudradar-monitoring/rport/share/query"
)

const (
	MinCredentialsLength = 3

	ErrCodeClientAuthSingleClient = "ERR_CODE_CLIENT_AUTH_SINGLE"
	ErrCodeClientAuthRO           = "ERR_CODE_CLIENT_AUTH_RO"

	ErrCodeClientAuthHasClient = "ERR_CODE_CLIENT_AUTH_HAS_CLIENT"
	ErrCodeClientAuthNotFound  = "ERR_CODE_CLIENT_AUTH_NOT_FOUND"
)

func (al *APIListener) handleGetClientAuth(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientAuthID := vars[routes.ParamClientAuthID]
	clientAuth, err := al.clientAuthProvider.Get(clientAuthID)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if clientAuth == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Client Auth with ID %q not found", clientAuthID))
		return
	}
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(clientAuth))
}
func (al *APIListener) handleGetClientsAuth(w http.ResponseWriter, req *http.Request) {
	options := query.NewOptions(req, nil, nil, nil)
	errs := query.ValidateListOptions(options, clientsauth.SupportedSorts, clientsauth.SupportedFilters, nil, &query.PaginationConfig{
		MaxLimit:     500,
		DefaultLimit: 50,
	})
	if errs != nil {
		al.jsonError(w, errs)
		return
	}
	rClients, count, err := al.clientAuthProvider.GetFiltered(options)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, &api.SuccessPayload{
		Data: rClients,
		Meta: api.NewMeta(count),
	})
}

func (al *APIListener) handlePostClientsAuth(w http.ResponseWriter, req *http.Request) {
	if !al.allowClientAuthWrite(w) {
		return
	}

	var newClient clientsauth.ClientAuth
	err := parseRequestBody(req.Body, &newClient)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	if len(newClient.ID) < MinCredentialsLength {
		al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid or missing ID.", fmt.Sprintf("Min size is %d.", MinCredentialsLength))
		return
	}

	if len(newClient.Password) < MinCredentialsLength {
		al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid or missing password.", fmt.Sprintf("Min size is %d.", MinCredentialsLength))
		return
	}

	added, err := al.clientAuthProvider.Add(&newClient)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if !added {
		al.jsonErrorResponseWithDetail(w, http.StatusConflict, ErrCodeAlreadyExist, fmt.Sprintf("Client Auth with ID %q already exist.", newClient.ID), "")
		return
	}

	al.auditLog.Entry(auditlog.ApplicationClientAuth, auditlog.ActionCreate).
		WithHTTPRequest(req).
		WithID(newClient.ID).
		Save()

	al.Infof("ClientAuth %q created.", newClient.ID)

	w.WriteHeader(http.StatusCreated)
}

func (al *APIListener) handleDeleteClientAuth(w http.ResponseWriter, req *http.Request) {
	if !al.allowClientAuthWrite(w) {
		return
	}

	vars := mux.Vars(req)
	clientAuthID := vars["client_auth_id"]
	if clientAuthID == "" {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeMissingRouteVar, "Missing 'client_auth_id' route param.")
		return
	}

	force := false
	forceStr := req.URL.Query().Get("force")
	if forceStr != "" {
		var err error
		force, err = strconv.ParseBool(forceStr)
		if err != nil {
			al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeInvalidRequest, fmt.Sprintf("Invalid force param %v.", forceStr))
			return
		}
	}

	existing, err := al.clientAuthProvider.Get(clientAuthID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if existing == nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusNotFound, ErrCodeClientAuthNotFound, fmt.Sprintf("Client Auth with ID=%q not found.", clientAuthID))
		return
	}

	allClients := al.clientService.GetAllByClientID(clientAuthID)
	if !force && len(allClients) > 0 {
		al.jsonErrorResponseWithErrCode(w, http.StatusConflict, ErrCodeClientAuthHasClient, fmt.Sprintf("Client Auth expected to have no active or disconnected bound client(s), got %d.", len(allClients)))
		return
	}

	for _, s := range allClients {
		if err := al.clientService.ForceDelete(s); err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
	}

	err = al.clientAuthProvider.Delete(clientAuthID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	al.Infof("ClientAuth %q deleted.", clientAuthID)

	al.auditLog.Entry(auditlog.ApplicationClientAuth, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(clientAuthID).
		WithRequest(map[string]interface{}{
			"force": force,
		}).
		Save()

	w.WriteHeader(http.StatusNoContent)
}

type clientsAuthMode string

const (
	clientsAuthModeRO = "Read Only"
	clientsAuthModeRW = "Read Write"
)

func (al *APIListener) getClientsAuthMode() clientsAuthMode {
	if al.isClientsAuthWriteable() {
		return clientsAuthModeRW
	}
	return clientsAuthModeRO
}

func (al *APIListener) isClientsAuthWriteable() bool {
	return al.clientAuthProvider.IsWriteable() && al.config.Server.AuthWrite
}

func (al *APIListener) allowClientAuthWrite(w http.ResponseWriter) bool {
	if !al.clientAuthProvider.IsWriteable() {
		al.jsonErrorResponseWithErrCode(w, http.StatusMethodNotAllowed, ErrCodeClientAuthSingleClient, "Client authentication is enabled only for a single user.")
		return false
	}

	if !al.config.Server.AuthWrite {
		al.jsonErrorResponseWithErrCode(w, http.StatusMethodNotAllowed, ErrCodeClientAuthRO, "Client authentication has been attached in read-only mode.")
		return false
	}

	return true
}
