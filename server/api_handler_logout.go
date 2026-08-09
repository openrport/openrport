package chserver

import (
	"fmt"
	"net/http"

	"github.com/openrport/openrport/server/bearer"
)

func (al *APIListener) handleDeleteLogout(w http.ResponseWriter, req *http.Request) {
	tokenStr, tokenProvided := al.getBearerTokenFromRequest(req)
	if tokenStr == "" || !tokenProvided {
		al.clearAuthCookie(w, req)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	tokenCtx, err := bearer.ParseToken(tokenStr, al.config.API.JWTSecret)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, fmt.Errorf("token is invalid: %v", err))
		return
	}

	if al.bannedUsers.IsBanned(tokenCtx.AppClaims.Username) {
		al.Errorf(
			"User %s is banned",
			tokenCtx.AppClaims.Username,
		)
		al.jsonErrorResponse(w, http.StatusInternalServerError, ErrTooManyRequests)
		return
	}

	valid, apiSession, err := bearer.ValidateBearerToken(
		req.Context(),
		tokenCtx,
		req.URL.Path,
		req.Method,
		al.apiSessions,
		al.Logger)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if !al.handleBannedIPs(req, valid) {
		return
	}
	if !valid {
		al.bannedUsers.Add(tokenCtx.AppClaims.Username)
		al.jsonErrorResponse(w, http.StatusBadRequest, fmt.Errorf("token is invalid or expired"))
		return
	}

	err = al.apiSessions.Delete(req.Context(), apiSession.SessionID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.clearAuthCookie(w, req)
	w.WriteHeader(http.StatusNoContent)
}
