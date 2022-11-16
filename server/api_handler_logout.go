package chserver

import (
	"fmt"
	"net/http"

	"github.com/cloudradar-monitoring/rport/server/bearer"
)

func (al *APIListener) handleDeleteLogout(w http.ResponseWriter, req *http.Request) {
	tokenStr, tokenProvided := bearer.GetBearerToken(req)
	if tokenStr == "" || !tokenProvided {
		// ban IP if it sends a lot of bad requests
		if !al.handleBannedIPs(req, false) {
			return
		}
		al.jsonErrorResponse(w, http.StatusBadRequest, fmt.Errorf("authorization Bearer token required"))
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

	err = al.apiSessions.Delete(req.Context(), apiSession.Token)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
