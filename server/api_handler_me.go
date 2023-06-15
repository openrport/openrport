package chserver

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/authorization"
	users "github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/auditlog"
	"github.com/realvnc-labs/rport/server/routes"
	chshare "github.com/realvnc-labs/rport/share"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/ptr"
	"github.com/realvnc-labs/rport/share/random"
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

	tr, cr := al.userService.GetEffectiveUserExtendedPermissions(user)

	me := UserPayload{
		Username:                 user.Username,
		Groups:                   user.Groups,
		TwoFASendTo:              user.TwoFASendTo,
		EffectiveUserPermissions: eup,
		TunnelsRestricted:        tr,
		CommandsRestricted:       cr,
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

func (al *APIListener) handleTokenGone(w http.ResponseWriter, req *http.Request) {
	al.jsonErrorResponseWithTitle(w, http.StatusGone, "use new token management on /me/tokens")
}

func (al *APIListener) handleGetToken(w http.ResponseWriter, req *http.Request) {
	user, err := al.getUserModel(req.Context())
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}
	type APITokenPayload struct {
		Prefix    string                      `json:"prefix" db:"prefix"`
		Name      string                      `json:"name" db:"name"`
		CreatedAt *time.Time                  `json:"created_at" db:"created_at"`
		ExpiresAt *time.Time                  `json:"expires_at" db:"expires_at"`
		Scope     authorization.APITokenScope `json:"scope" db:"scope"`
	}

	apitokenset, err := al.tokenManager.GetAll(req.Context(), user.Username)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	apiTokenToSend := make([]APITokenPayload, 0, len(apitokenset))
	for _, at := range apitokenset {
		apiTokenToSend = append(apiTokenToSend,
			APITokenPayload{
				Prefix:    at.Prefix,
				Name:      at.Name,
				CreatedAt: at.CreatedAt,
				ExpiresAt: at.ExpiresAt,
				Scope:     at.Scope,
			})
	}

	response := api.NewSuccessPayload(apiTokenToSend)
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handlePostToken(w http.ResponseWriter, req *http.Request) {
	user, err := al.getUserModel(req.Context())
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}
	var r struct {
		Scope     authorization.APITokenScope `json:"scope"`
		Name      string                      `json:"name"`
		ExpiresAt *time.Time                  `json:"expires_at"`
	}
	err = parseRequestBody(req.Body, &r)
	if err != nil {
		al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, "", "Invalid JSON data", err.Error())
		return
	}

	if !authorization.IsValidScope(r.Scope) {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "missing or invalid scope.")
		return
	}

	if len(r.Name) == 0 || len(r.Name) >= 250 {
		al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, "", "missing or invalid name.", "field name is required, 250 characters max")
		return
	}

	if r.Scope == authorization.APITokenClientsAuth && !user.IsAdmin() {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "current user should belong to Administrators group to create a token with this scope")
		return
	}

	createdAt := ptr.Time(time.Now().Truncate(time.Second).UTC())
	if r.ExpiresAt == nil {
		r.ExpiresAt = ptr.Time(createdAt.AddDate(1 /* year */, 0, 0)) // expiry date default is creation date + one year
	}

	newTokenClear, err := random.UUID4()
	if err != nil {
		al.jsonError(w, err)
		return
	}
	newPrefix := random.AlphaNum(authorization.APITokenPrefixLength)

	tokenHashStr, err := users.GenerateTokenHash(newTokenClear)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	newAPIToken := &authorization.APIToken{
		Username:  user.Username,
		Prefix:    newPrefix,
		Name:      r.Name,
		Scope:     r.Scope,
		CreatedAt: createdAt,
		ExpiresAt: r.ExpiresAt,
		Token:     tokenHashStr,
	}
	err = al.tokenManager.Create(req.Context(), newAPIToken)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, err.Error())
		return
	}

	al.auditLog.Entry(auditlog.ApplicationAuthUserMeToken, auditlog.ActionCreate).
		WithHTTPRequest(req).
		WithID(fmt.Sprintf("[%s,%s]", user.Username, newPrefix)).
		Save()

	al.Debugf("APIToken [%s] is created for user [%s].", newPrefix, user.Username)

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(
		authorization.APIToken{
			ExpiresAt: r.ExpiresAt,
			Scope:     r.Scope,
			Token:     fmt.Sprintf("%s_%s", newPrefix, newTokenClear),
			Prefix:    newPrefix,
		}))
}

func (al *APIListener) handlePutToken(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	prefix := vars[routes.ParamTokenPrefix]
	if len(prefix) != authorization.APITokenPrefixLength {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "missing or invalid token prefix.")
		return
	}

	user, err := al.getUserModel(req.Context())
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}

	var r struct {
		ExpiresAt *time.Time `json:"expires_at"`
		Name      string     `json:"name"`
	}
	err = parseRequestBody(req.Body, &r)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	updAPIToken := &authorization.APIToken{
		Username:  user.Username,
		Prefix:    prefix,
		Name:      r.Name,
		ExpiresAt: r.ExpiresAt,
	}
	err = al.tokenManager.Save(req.Context(), updAPIToken)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, err.Error())
		return
	}

	al.auditLog.Entry(auditlog.ApplicationAuthUserMeToken, auditlog.ActionUpdate).
		WithHTTPRequest(req).
		WithID(user.Username).
		WithRequest(r).
		Save()

	al.Debugf("APIToken [%s] is updated for user [%s].", prefix, user.Username)
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(
		authorization.APIToken{
			ExpiresAt: r.ExpiresAt,
			Prefix:    prefix,
			Name:      r.Name,
		}))
}

func (al *APIListener) handleDeleteToken(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	prefix := vars[routes.ParamTokenPrefix]
	if len(prefix) != authorization.APITokenPrefixLength {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "missing or invalid token prefix.")
		return
	}

	user, err := al.getUserModel(req.Context())
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}

	err = al.tokenManager.Delete(req.Context(), user.Username, prefix)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	al.auditLog.Entry(auditlog.ApplicationAuthUserMeToken, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(user.Username).
		WithRequest(req).
		Save()

	al.Debugf("APIToken [%s] is deleted for user [%s].", prefix, user.Username)
	w.WriteHeader(http.StatusNoContent)
}
