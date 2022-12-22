package chserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/authorization"
	users "github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/random"
	"golang.org/x/crypto/bcrypt"
)

// handleGetMe returns the currently logged-in user and the groups the user belongs to.
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
	eup, err := al.userService.GetEffectiveUserPermissions(user)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	me := UserPayload{
		Username:                 user.Username,
		Groups:                   user.Groups,
		TwoFASendTo:              user.TwoFASendTo,
		EffectiveUserPermissions: eup,
		GroupPermissionsEnabled:  al.userService.SupportsGroupPermissions(),
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

func (al *APIListener) handleManageAPIToken(w http.ResponseWriter, req *http.Request, user *users.User, action string) {
	if action == "delete" {
		var r struct {
			Prefix string `json:"prefix"`
		}
		err := parseRequestBody(req.Body, &r)
		if err != nil {
			al.jsonError(w, err)
			return
		}

		if len(r.Prefix) != 8 {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "missing or invalid token prefix.")
			return
		}

		err = al.tokenManager.Delete(req.Context(), user.Username, r.Prefix)
		if err != nil {
			al.jsonError(w, err)
			return
		}
		al.auditLog.Entry(auditlog.ApplicationAuthUserMeToken, auditlog.ActionDelete).
			WithHTTPRequest(req).
			WithID(user.Username).
			Save()

		al.Debugf("APIToken %s is deleted for user [%s].", r.Prefix, user.Username)
		w.WriteHeader(http.StatusNoContent)
	}
	if action == "create" {
		// EDTODO: Reject the creation of a `clients-auth`-token if the user is not in the admin group.

		var r struct {
			Scope string `json:"scope"`
		}
		err := parseRequestBody(req.Body, &r)
		if err != nil {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "missing body with scope.")
			return
		}

		if !authorization.IsValidScope(r.Scope) {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "missing or invalid scope.")
			return
		}

		newTokenClear, err := random.UUID4()
		if err != nil {
			al.jsonError(w, err)
			return
		}
		newPrefix := random.AlphaNum(8)

		// token creation
		tokenHash, err := bcrypt.GenerateFromPassword([]byte(newTokenClear), bcrypt.DefaultCost)
		if err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		tokenHashStr := strings.Replace(string(tokenHash), users.HtpasswdBcryptAltPrefix, users.HtpasswdBcryptPrefix, 1)

		newAPIToken := &authorization.APIToken{
			Username: user.Username,
			Prefix:   newPrefix,
			Scope:    r.Scope,
			Token:    tokenHashStr,
		}
		err = al.tokenManager.Create(req.Context(), newAPIToken)
		if err != nil {
			al.jsonError(w, err)
			return
		}

		al.auditLog.Entry(auditlog.ApplicationAuthUserMeToken, auditlog.ActionCreate).
			WithHTTPRequest(req).
			WithID(user.Username).
			Save()

		al.Debugf("APIToken [%s] is created for user [%s].", newPrefix, user.Username)
		al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(
			authorization.APIToken{
				Prefix: newPrefix,
				Scope:  r.Scope,
				Token:  newTokenClear,
			}))
	}
	if action == "update" {
		var r struct {
			Prefix    string     `json:"prefix"`
			ExpiresAt *time.Time `json:"expires_at"`
		}
		err := parseRequestBody(req.Body, &r)
		if err != nil {
			al.jsonError(w, err)
			return
		}

		updAPIToken := &authorization.APIToken{
			Username:  user.Username,
			Prefix:    r.Prefix,
			ExpiresAt: r.ExpiresAt,
		}
		err = al.tokenManager.Save(req.Context(), updAPIToken)
		if err != nil {
			al.jsonError(w, err)
			return
		}

		al.auditLog.Entry(auditlog.ApplicationAuthUserMeToken, auditlog.ActionUpdate).
			WithHTTPRequest(req).
			WithID(user.Username).
			WithID(r.Prefix).
			Save()

		al.Debugf("APIToken [%s] is updated for user [%s].", r.Prefix, user.Username)
		al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(updAPIToken))
	}
	if action == "list" {
		type APITokenPayload struct {
			Prefix    string     `json:"prefix" db:"token"`
			CreatedAt *time.Time `json:"created_at" db:"created_at"`
			ExpiresAt *time.Time `json:"expires_at" db:"expires_at"`
			Scope     string     `json:"scope" db:"scope"`
		}

		apitokenset, err := al.tokenManager.GetAll(req.Context(), user.Username)
		if err != nil {
			al.jsonError(w, err)
			return
		}

		apiTokenToSend := make([]APITokenPayload, 0, len(apitokenset))
		for i := range apitokenset {
			at := apitokenset[i]
			apiTokenToSend = append(apiTokenToSend,
				APITokenPayload{
					Prefix:    at.Prefix,
					CreatedAt: at.CreatedAt,
					ExpiresAt: at.ExpiresAt,
					Scope:     at.Scope,
				})
		}

		response := api.NewSuccessPayload(apiTokenToSend)
		al.writeJSONResponse(w, http.StatusOK, response)
		return
	}
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

func (al *APIListener) handleGetToken(w http.ResponseWriter, req *http.Request) {
	al.handleManageCurUserAPIToken(w, req, "list")
}

func (al *APIListener) handlePostToken(w http.ResponseWriter, req *http.Request) {
	al.handleManageCurUserAPIToken(w, req, "create")
}

func (al *APIListener) handlePutToken(w http.ResponseWriter, req *http.Request) {
	al.handleManageCurUserAPIToken(w, req, "update")
}

func (al *APIListener) handleDeleteToken(w http.ResponseWriter, req *http.Request) {
	al.handleManageCurUserAPIToken(w, req, "delete")
}

func (al *APIListener) handleManageCurUserAPIToken(w http.ResponseWriter, req *http.Request, action string) {
	user, err := al.getUserModel(req.Context())
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}
	al.handleManageAPIToken(w, req, user, action)
}
