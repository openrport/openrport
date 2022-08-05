package chserver

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cloudradar-monitoring/rport/server/api"
	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/jobs/schedule"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
)

func (al *APIListener) handleListSchedules(w http.ResponseWriter, req *http.Request) {
	items, err := al.scheduleManager.List(req.Context(), req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(items))
}

func (al *APIListener) handlePostSchedules(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var scheduleInput schedule.Schedule
	err := parseRequestBody(req.Body, &scheduleInput)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	orderedClients, _, err := al.getOrderedClientsWithValidation(ctx, &scheduleInput)
	if err != nil {
		al.jsonErrorResponseWithAPIError(w, err)
		return
	}

	err = al.clientService.CheckClientsAccess(orderedClients, curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	storedValue, err := al.scheduleManager.Create(ctx, &scheduleInput, curUser.GetUsername())
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

	var scheduleInput schedule.Schedule
	err := parseRequestBody(req.Body, &scheduleInput)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	orderedClients, _, err := al.getOrderedClientsWithValidation(ctx, &scheduleInput)
	if err != nil {
		al.jsonErrorResponseWithAPIError(w, err)
		return
	}

	err = al.clientService.CheckClientsAccess(orderedClients, curUser)
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
