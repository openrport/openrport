package chserver

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/realvnc-labs/rport/server/api"
	errors2 "github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/server/api/jobs/schedule"
	"github.com/realvnc-labs/rport/server/auditlog"
	"github.com/realvnc-labs/rport/server/clients/clientdata"
)

func (al *APIListener) handleListSchedules(w http.ResponseWriter, req *http.Request) {
	items, err := al.scheduleManager.List(req.Context(), req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(items))
}

func (al *APIListener) prepareHandleSchedules(req *http.Request) (schedule.Schedule, string, []*clientdata.Client, error) {
	var scheduleInput schedule.Schedule
	var orderedClients []*clientdata.Client
	var username string
	ctx := req.Context()
	err := parseRequestBody(req.Body, &scheduleInput)
	if err != nil {
		return scheduleInput, username, orderedClients, err
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		return scheduleInput, username, orderedClients, err
	}
	username = curUser.GetUsername()

	orderedClients, _, err = al.getOrderedClientsWithValidation(ctx, &scheduleInput)
	if err != nil {
		return scheduleInput, username, orderedClients, err
	}

	clientGroups, err := al.clientGroupProvider.GetAll(ctx)
	if err != nil {
		return scheduleInput, username, orderedClients, err
	}
	err = al.clientService.CheckClientsAccess(orderedClients, curUser, clientGroups)
	if err != nil {
		return scheduleInput, username, orderedClients, err
	}
	return scheduleInput, username, orderedClients, nil
}

func (al *APIListener) handlePostSchedules(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	scheduleInput, username, orderedClients, err := al.prepareHandleSchedules(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	storedValue, err := al.scheduleManager.Create(ctx, &scheduleInput, username)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationSchedule, auditlog.ActionCreate).
		WithHTTPRequest(req).
		WithRequest(scheduleInput).
		WithResponse(storedValue).
		WithID(storedValue.ID).
		SaveForMultipleClients(orderedClients)

	al.writeJSONResponse(w, http.StatusCreated, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleUpdateSchedule(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	idStr, ok := vars["schedule_id"]
	if !ok {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Schedule ID is not provided")
		return
	}

	scheduleInput, _, orderedClients, err := al.prepareHandleSchedules(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	storedValue, err := al.scheduleManager.Update(ctx, idStr, &scheduleInput)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationSchedule, auditlog.ActionUpdate).
		WithHTTPRequest(req).
		WithRequest(scheduleInput).
		WithResponse(storedValue).
		WithID(idStr).
		SaveForMultipleClients(orderedClients)

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleGetSchedule(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars["schedule_id"]
	if idStr == "" {
		al.jsonError(w, errors2.APIError{
			Err:        errors.New("empty schedule id provided"),
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	foundSchedule, err := al.scheduleManager.Get(req.Context(), idStr)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if foundSchedule == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Cannot find a schedule by the provided id: %s", idStr))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(foundSchedule))
}

func (al *APIListener) handleDeleteSchedule(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars["schedule_id"]
	if idStr == "" {
		al.jsonError(w, errors2.APIError{
			Err:        errors.New("empty schedule id provided"),
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	err := al.scheduleManager.Delete(req.Context(), idStr)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationSchedule, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(idStr).
		Save()

	w.WriteHeader(http.StatusNoContent)
}
