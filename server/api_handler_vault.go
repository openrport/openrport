package chserver

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/cloudradar-monitoring/rport/server/api"
	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/vault"
)

func (al *APIListener) handleGetVaultStatus(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	st, err := al.vaultManager.Status(ctx)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(st))
}

func (al *APIListener) handleVaultUnlock(w http.ResponseWriter, req *http.Request) {
	var passReq vault.PassRequest
	err := parseRequestBody(req.Body, &passReq)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.vaultManager.UnLock(req.Context(), passReq.Password)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationVault, "unlock").
		WithHTTPRequest(req).
		Save()

	w.WriteHeader(http.StatusCreated)
}

func (al *APIListener) handleVaultLock(w http.ResponseWriter, req *http.Request) {
	err := al.vaultManager.Lock(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationVault, "lock").
		WithHTTPRequest(req).
		Save()

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) handleVaultInit(w http.ResponseWriter, req *http.Request) {
	var passReq vault.PassRequest
	err := parseRequestBody(req.Body, &passReq)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.vaultManager.Init(req.Context(), passReq.Password)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationVault, "init").
		WithHTTPRequest(req).
		Save()

	w.WriteHeader(http.StatusCreated)
}

func (al *APIListener) handleListVaultValues(w http.ResponseWriter, req *http.Request) {
	items, err := al.vaultManager.List(req.Context(), req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(items))
}

func (al *APIListener) readIntParam(paramName string, req *http.Request) (int, error) {
	vars := mux.Vars(req)
	idStr, ok := vars[paramName]
	if !ok {
		return 0, nil
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("Non-numeric integer value provided: %s for param %s", idStr, paramName)
	}

	return id, nil
}

func (al *APIListener) handleReadVaultValue(w http.ResponseWriter, req *http.Request) {
	id, err := al.readIntParam(routeParamVaultValueID, req)
	if err != nil {
		al.jsonError(w, errors2.APIError{
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}
	if id == 0 {
		al.jsonError(w, errors2.APIError{
			Err:        fmt.Errorf("missing %q route param", routeParamVaultValueID),
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	storedValue, found, err := al.vaultManager.GetOne(req.Context(), id, curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if !found {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Cannot find a vault value by the provided id: %d", id))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleVaultStoreValue(w http.ResponseWriter, req *http.Request) {
	id, err := al.readIntParam(routeParamVaultValueID, req)
	if err != nil {
		al.jsonError(w, errors2.APIError{
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	var vaultKeyValue vault.InputValue
	err = parseRequestBody(req.Body, &vaultKeyValue)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	storedValue, err := al.vaultManager.Store(req.Context(), int64(id), &vaultKeyValue, curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	status := http.StatusOK

	vaultKeyValue.Value = ""
	if id == 0 {
		al.auditLog.Entry(auditlog.ApplicationVault, auditlog.ActionCreate).
			WithHTTPRequest(req).
			WithID(storedValue.ID).
			WithClientID(vaultKeyValue.ClientID).
			WithRequest(vaultKeyValue).
			Save()

		w.WriteHeader(http.StatusCreated)
	} else {
		al.auditLog.Entry(auditlog.ApplicationVault, auditlog.ActionUpdate).
			WithHTTPRequest(req).
			WithID(id).
			WithClientID(vaultKeyValue.ClientID).
			WithRequest(vaultKeyValue).
			Save()
	}

	al.writeJSONResponse(w, status, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleVaultDeleteValue(w http.ResponseWriter, req *http.Request) {
	id, err := al.readIntParam(routeParamVaultValueID, req)
	if err != nil {
		al.jsonError(w, errors2.APIError{
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}
	if id == 0 {
		al.jsonError(w, errors2.APIError{
			Err:        fmt.Errorf("missing %q route param", routeParamVaultValueID),
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	storedValue, found, err := al.vaultManager.GetOne(req.Context(), id, curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if !found {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Cannot find a vault value by the provided id: %d", id))
		return
	}

	err = al.vaultManager.Delete(req.Context(), id, curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationVault, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(id).
		WithClientID(storedValue.ClientID).
		Save()

	w.WriteHeader(http.StatusNoContent)
}
