package chserver

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	rportplus "github.com/realvnc-labs/rport/plus"
	extperm "github.com/realvnc-labs/rport/plus/capabilities/extendedpermission"
	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/auditlog"
	"github.com/realvnc-labs/rport/server/routes"
)

var (
	ErrMissingUserIDParam    = errors.New("missing user id param")
	ErrMissingSessionIDParam = errors.New("missing session id param")
)

type UserPayload struct {
	Username                 string                     `json:"username"`
	PasswordExpired          bool                       `json:"password_expired"`
	Groups                   []string                   `json:"groups"`
	TwoFASendTo              string                     `json:"two_fa_send_to"`
	EffectiveUserPermissions map[string]bool            `json:"effective_user_permissions"`
	TunnelsRestricted        []extperm.PermissionParams `json:"tunnels_restricted" db:"tunnels_restricted"`
	CommandsRestricted       []extperm.PermissionParams `json:"commands_restricted" db:"commands_restricted"`
	GroupPermissionsEnabled  bool                       `json:"group_permissions_enabled"`
}

func (al *APIListener) handleGetUsers(w http.ResponseWriter, req *http.Request) {
	usrs, err := al.userService.GetAll()
	if err != nil {
		al.jsonError(w, err)
		return
	}

	usersToSend := make([]UserPayload, 0, len(usrs))
	for _, user := range usrs {
		payload := UserPayload{
			Username:    user.Username,
			Groups:      user.Groups,
			TwoFASendTo: user.TwoFASendTo,
		}
		if user.PasswordExpired != nil {
			payload.PasswordExpired = *user.PasswordExpired
		}
		usersToSend = append(usersToSend, payload)
	}

	response := api.NewSuccessPayload(usersToSend)
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleChangeUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	// if the username is part of the route params then we're updating the user, otherwise creating a new user
	userID, userIDExists := vars[routes.ParamUserID]
	if !userIDExists {
		userID = ""

		err := al.checkUserCount()
		if err != nil {
			al.jsonError(w, err)
			return
		}
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

	if user.PasswordExpired != nil && *user.PasswordExpired {
		// this user password was just set to expired, need to kill all his/her sessions
		ctx := req.Context()
		err := al.apiSessions.DeleteAllByUser(ctx, userID)
		if err != nil {
			titleMsg := fmt.Sprintf("password expired, unable to delete all sessions for user \"%s\"", userID)
			al.jsonErrorResponseWithDetail(w, http.StatusInternalServerError, "Unable to delete all User's sessions", titleMsg, err.Error())
			return
		}
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

func (al *APIListener) checkUserCount() (err error) {
	maxUsers := al.getMaxUsers()

	if maxUsers > 0 {
		users, err := al.userService.GetAll()
		if err != nil {
			return err
		}

		if len(users) >= maxUsers {
			return errors.New("failed to create user. max user limit reached. please upgrade your license for additional users")
		}
	}

	return nil
}

func (al *APIListener) getMaxUsers() (maxUsers int) {
	if rportplus.IsPlusEnabled(al.config.PlusConfig) {
		maxUsers = al.Server.plusManager.GetLicenseCapabilityEx().GetMaxUsers()
	}
	return maxUsers
}

func (al *APIListener) handleDeleteUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID, userIDExists := vars[routes.ParamUserID]
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
	userID, userIDProvided := vars[routes.ParamUserID]
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

func (al *APIListener) handleGetUserAPISessions(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID, userIDExists := vars[routes.ParamUserID]
	if !userIDExists {
		al.jsonError(w, ErrMissingUserIDParam)
		return
	}

	ctx := req.Context()

	userSessions, err := al.apiSessions.GetAllByUser(ctx, userID)
	if err != nil {
		titleMsg := fmt.Sprintf("unable to get sessions for user \"%s\"", userID)
		al.jsonErrorResponseWithDetail(w, http.StatusInternalServerError, "", titleMsg, err.Error())
		return
	}

	response := api.NewSuccessPayload(userSessions)
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleDeleteUserAPISession(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID, userIDExists := vars[routes.ParamUserID]
	if !userIDExists {
		al.jsonError(w, ErrMissingUserIDParam)
		return
	}

	sessionID, sessionIDExists := vars[routes.ParamSessionID]
	if !sessionIDExists {
		al.jsonError(w, ErrMissingUserIDParam)
		return
	}

	ctx := req.Context()

	sessID, err := strconv.ParseInt(sessionID, 10, 64)
	if err != nil {
		titleMsg := fmt.Sprintf("unable to parse session ID \"%s\" for user \"%s\"", sessionID, userID)
		al.jsonErrorResponseWithDetail(w, http.StatusInternalServerError, "", titleMsg, err.Error())
		return
	}

	err = al.apiSessions.DeleteByID(ctx, userID, sessID)
	if err != nil {
		titleMsg := fmt.Sprintf("unable to delete session \"%s\" for user \"%s\"", sessionID, userID)
		al.jsonErrorResponseWithDetail(w, http.StatusInternalServerError, "", titleMsg, err.Error())
		return
	}

	al.auditLog.Entry(auditlog.ApplicationAuthAPISession, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(userID).
		Save()

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) handleDeleteAllUserAPISessions(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID, userIDExists := vars[routes.ParamUserID]
	if !userIDExists {
		al.jsonError(w, ErrMissingUserIDParam)
		return
	}

	ctx := req.Context()

	err := al.apiSessions.DeleteAllByUser(ctx, userID)
	if err != nil {
		titleMsg := fmt.Sprintf("unable to delete all sessions for user \"%s\"", userID)
		al.jsonErrorResponseWithDetail(w, http.StatusInternalServerError, "", titleMsg, err.Error())
		return
	}

	al.auditLog.Entry(auditlog.ApplicationAuthAPISessions, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(userID).
		Save()

	w.WriteHeader(http.StatusNoContent)
}
