package chserver

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
)

type UserPayload struct {
	Username    string   `json:"username"`
	Groups      []string `json:"groups"`
	TwoFASendTo string   `json:"two_fa_send_to"`
}

func (al *APIListener) handleGetUsers(w http.ResponseWriter, req *http.Request) {
	usrs, err := al.userService.GetAll()
	if err != nil {
		al.jsonError(w, err)
		return
	}

	usersToSend := make([]UserPayload, 0, len(usrs))
	for i := range usrs {
		user := usrs[i]
		usersToSend = append(usersToSend, UserPayload{
			Username:    user.Username,
			Groups:      user.Groups,
			TwoFASendTo: user.TwoFASendTo,
		})
	}

	response := api.NewSuccessPayload(usersToSend)
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleChangeUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID, userIDExists := vars[routeParamUserID]
	if !userIDExists {
		userID = ""
	}

	var user users.User
	err := parseRequestBody(req.Body, &user)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	if err := al.userService.Change(&user, userID); err != nil {
		al.jsonError(w, err)
		return
	}

	if userIDExists {
		al.auditLog.Entry(auditlog.ApplicationAuthUser, auditlog.ActionUpdate).
			WithHTTPRequest(req).
			WithID(userID).
			Save()

		al.Debugf("User [%s] updated.", userID)
		w.WriteHeader(http.StatusNoContent)
	} else {
		al.auditLog.Entry(auditlog.ApplicationAuthUser, auditlog.ActionCreate).
			WithHTTPRequest(req).
			Save()

		al.Debugf("User [%s] created.", user.Username)
		w.WriteHeader(http.StatusCreated)
	}
}

func (al *APIListener) handleDeleteUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID, userIDExists := vars[routeParamUserID]
	if !userIDExists {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Empty user id provided")
		return
	}

	if err := al.userService.Delete(userID); err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry("user", auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(userID).
		Save()

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("User [%s] deleted.", userID)
}

func (al *APIListener) handleDeleteUsersTotP(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID, userIDProvided := vars[routeParamUserID]
	if !userIDProvided {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Empty user id provided")
		return
	}

	user, err := al.userService.GetByUsername(userID)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}

	al.auditLog.Entry(auditlog.ApplicationAuthUserTotP, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(userID).
		Save()

	al.handleManageTotP(w, req, user, "delete")
}
