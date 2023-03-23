package chserver

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/command"
	errors2 "github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/server/auditlog"
	"github.com/realvnc-labs/rport/server/routes"
	"github.com/realvnc-labs/rport/server/script"
)

func (al *APIListener) handleListScripts(w http.ResponseWriter, req *http.Request) {
	items, count, err := al.scriptManager.List(req.Context(), req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, &api.SuccessPayload{
		Data: items,
		Meta: api.NewMeta(count),
	})
}

func (al *APIListener) handleScriptCreate(w http.ResponseWriter, req *http.Request) {
	var scriptInput script.InputScript
	err := parseRequestBody(req.Body, &scriptInput)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	curUsername := api.GetUser(req.Context(), al.Logger)
	if curUsername == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	storedValue, err := al.scriptManager.Create(req.Context(), &scriptInput, curUsername)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationLibraryScript, auditlog.ActionCreate).
		WithHTTPRequest(req).
		WithRequest(scriptInput).
		WithResponse(storedValue).
		WithID(storedValue.ID).
		Save()

	w.WriteHeader(http.StatusCreated)

	al.writeJSONResponse(w, http.StatusCreated, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleScriptUpdate(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr, ok := vars[routes.ParamScriptValueID]
	if !ok {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Script ID is not provided")
		return
	}

	curUsername := api.GetUser(req.Context(), al.Logger)
	if curUsername == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var scriptInput script.InputScript
	err := parseRequestBody(req.Body, &scriptInput)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	storedValue, err := al.scriptManager.Update(req.Context(), idStr, &scriptInput, curUsername)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationLibraryScript, auditlog.ActionUpdate).
		WithHTTPRequest(req).
		WithRequest(scriptInput).
		WithResponse(storedValue).
		WithID(idStr).
		Save()

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleReadScript(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars[routes.ParamScriptValueID]
	if idStr == "" {
		al.jsonError(w, errors2.APIError{
			Err:        errors.New("empty script id provided"),
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	foundScript, found, err := al.scriptManager.GetOne(req.Context(), req, idStr)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if !found {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Cannot find a script by the provided id: %s", idStr))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(foundScript))
}

func (al *APIListener) handleDeleteScript(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars[routes.ParamScriptValueID]
	if idStr == "" {
		al.jsonError(w, errors2.APIError{
			Err:        errors.New("empty script id provided"),
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	err := al.scriptManager.Delete(req.Context(), idStr)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationLibraryScript, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(idStr).
		Save()

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) handleListCommands(w http.ResponseWriter, req *http.Request) {
	items, count, err := al.commandManager.List(req.Context(), req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.SuccessPayload{
		Data: items,
		Meta: api.NewMeta(count),
	})
}

func (al *APIListener) handleCommandCreate(w http.ResponseWriter, req *http.Request) {
	var commandInput command.InputCommand
	err := parseRequestBody(req.Body, &commandInput)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	curUsername := api.GetUser(req.Context(), al.Logger)
	if curUsername == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	storedValue, err := al.commandManager.Create(req.Context(), &commandInput, curUsername)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationLibraryCommand, auditlog.ActionCreate).
		WithHTTPRequest(req).
		WithRequest(commandInput).
		WithResponse(storedValue).
		WithID(storedValue.ID).
		Save()

	al.writeJSONResponse(w, http.StatusCreated, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleCommandUpdate(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr, ok := vars[routes.ParamCommandValueID]
	if !ok {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Command ID is not provided")
		return
	}

	curUsername := api.GetUser(req.Context(), al.Logger)
	if curUsername == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var commandInput command.InputCommand
	err := parseRequestBody(req.Body, &commandInput)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	storedValue, err := al.commandManager.Update(req.Context(), idStr, &commandInput, curUsername)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationLibraryCommand, auditlog.ActionUpdate).
		WithHTTPRequest(req).
		WithRequest(commandInput).
		WithResponse(storedValue).
		WithID(idStr).
		Save()

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleReadCommand(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars[routes.ParamCommandValueID]
	if idStr == "" {
		al.jsonError(w, errors2.APIError{
			Err:        errors.New("empty command id provided"),
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	foundScript, found, err := al.commandManager.GetOne(req.Context(), req, idStr)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if !found {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Cannot find a command by the provided id: %s", idStr))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(foundScript))
}

func (al *APIListener) handleDeleteCommand(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars[routes.ParamCommandValueID]
	if idStr == "" {
		al.jsonError(w, errors2.APIError{
			Err:        errors.New("empty command id provided"),
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	err := al.commandManager.Delete(req.Context(), idStr)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationLibraryCommand, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(idStr).
		Save()

	w.WriteHeader(http.StatusNoContent)
}
