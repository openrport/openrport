package chserver

import (
	"errors"
	"net/http"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/random"
)

// handleGetMe returns the currently logged in user and the groups the user belongs to.
func (al *APIListener) handleGetMe(w http.ResponseWriter, req *http.Request) {
	user, err := al.getUserModel(req.Context())
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}

	me := UserPayload{
		Username:    user.Username,
		Groups:      user.Groups,
		TwoFASendTo: user.TwoFASendTo,
	}
	response := api.NewSuccessPayload(me)
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleGetTotP(w http.ResponseWriter, req *http.Request) {
	user, err := al.getUserModel(req.Context())
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}

	totP, err := GetUsersTotPCode(user)
	if err != nil {
		al.Logf(logger.LogLevelError, "failed to get TotP secret: %v", err)
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if totP == nil || totP.Secret == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "time based one time secret key should be generated for this user")
		return
	}

	al.writeJSONResponse(w, http.StatusOK, totP)
}

func (al *APIListener) handlePostTotP(w http.ResponseWriter, req *http.Request) {
	al.handleManageCurUserTotP(w, req, "create")
}

func (al *APIListener) handleDeleteTotP(w http.ResponseWriter, req *http.Request) {
	al.handleManageCurUserTotP(w, req, "delete")
}

func (al *APIListener) handleManageCurUserTotP(w http.ResponseWriter, req *http.Request, action string) {
	user, err := al.getUserModel(req.Context())
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}
	al.handleManageTotP(w, req, user, action)
}

func (al *APIListener) handleManageTotP(w http.ResponseWriter, req *http.Request, user *users.User, action string) {
	totP := &TotP{}
	if action == "create" {
		existingTotP, err := GetUsersTotPCode(user)
		if err != nil {
			al.Logf(logger.LogLevelError, "failed to read TotP secret for user %s: %v", user.Username, err)
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if existingTotP != nil {
			err := errors.New("cannot create new totP secret when another one already exists")
			al.Logf(logger.LogLevelError, err.Error())
			al.jsonErrorResponse(w, http.StatusConflict, err)
			return
		}

		totP, err = GenerateTotPSecretKey(&TotPInput{
			Issuer:      user.Username,
			AccountName: al.config.API.TotPAccountName,
		})
		if err != nil {
			al.Logf(logger.LogLevelError, "failed to generate TotP secret for user %s: %v", user.Username, err)
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
	}

	userDataToChange := &users.User{}

	StoreTotPCodeInUser(userDataToChange, totP)

	if userDataToChange.TotP == "" {
		userDataToChange.TotP = " "
	}

	if err := al.userService.Change(userDataToChange, user.Username); err != nil {
		al.jsonError(w, err)
		return
	}

	if action == "create" {
		al.auditLog.Entry(auditlog.ApplicationAuthUserTotP, auditlog.ActionCreate).
			WithHTTPRequest(req).
			WithID(userDataToChange.Username).
			Save()

		al.Debugf("Users time based one time secret is created for user [%s].", user.Username)
		al.writeJSONResponse(w, http.StatusOK, totP)
	} else if action == "delete" {
		al.auditLog.Entry(auditlog.ApplicationAuthUserTotP, auditlog.ActionDelete).
			WithHTTPRequest(req).
			WithID(userDataToChange.Username).
			Save()

		al.Debugf("Users time based one time secret is deleted for user [%s].", user.Username)
		w.WriteHeader(http.StatusNoContent)
	}
}

type changeMeRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	OldPassword string `json:"old_password"`
	TwoFASendTo string `json:"two_fa_send_to"`
}

func (al *APIListener) handleChangeMe(w http.ResponseWriter, req *http.Request) {
	var r changeMeRequest
	err := parseRequestBody(req.Body, &r)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	if r.Password != "" {
		if r.OldPassword == "" {
			al.jsonErrorResponseWithTitle(w, http.StatusForbidden, "Missing old password.")
			return
		}

		if !verifyPassword(curUser.Password, r.OldPassword) {
			al.jsonErrorResponseWithTitle(w, http.StatusForbidden, "Incorrect old password.")
			return
		}
	}

	if err := al.userService.Change(&users.User{
		Username:    r.Username,
		Password:    r.Password,
		TwoFASendTo: r.TwoFASendTo,
	}, curUser.Username); err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationAuthUserMe, auditlog.ActionUpdate).
		WithHTTPRequest(req).
		Save()

	w.WriteHeader(http.StatusNoContent)
}

// handleGetIP handles GET /me/ip
func (al *APIListener) handleGetIP(w http.ResponseWriter, req *http.Request) {
	ipResp := struct {
		IP string `json:"ip"`
	}{
		IP: chshare.RemoteIP(req),
	}
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(ipResp))
}

type postTokenResponse struct {
	Token string `json:"token"`
}

// handlePostToken handles POST /me/token
func (al *APIListener) handlePostToken(w http.ResponseWriter, req *http.Request) {
	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	newToken, err := random.UUID4()
	if err != nil {
		al.jsonError(w, err)
		return
	}

	if err := al.userService.Change(&users.User{
		Token: &newToken,
	}, curUser.Username); err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationAuthUserMeToken, auditlog.ActionCreate).
		WithHTTPRequest(req).
		Save()

	resp := postTokenResponse{
		Token: newToken,
	}
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(resp))
}

// handleDeleteToken handles DELETE /me/token
func (al *APIListener) handleDeleteToken(w http.ResponseWriter, req *http.Request) {
	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	noToken := ""
	if err := al.userService.Change(&users.User{
		Token: &noToken,
	}, curUser.Username); err != nil {
		al.jsonError(w, err)
		return
	}
	al.auditLog.Entry(auditlog.ApplicationAuthUserMeToken, auditlog.ActionDelete).
		WithHTTPRequest(req).
		Save()

	w.WriteHeader(http.StatusNoContent)
}
