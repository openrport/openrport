package chserver

import (
	"errors"
	"fmt"
	"net/http"
)

func (al *APIListener) handleDeleteLogout(w http.ResponseWriter, req *http.Request) {
	tokenStr, tokenProvided := getBearerToken(req)
	if tokenStr == "" || !tokenProvided {
		// ban IP if it sends a lot of bad requests
		if !al.handleBannedIPs(req, false) {
			return
		}
		al.jsonErrorResponse(w, http.StatusBadRequest, fmt.Errorf("authorization Bearer token required"))
		return
	}

	token, err := al.parseToken(tokenStr)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, fmt.Errorf("token is invalid: %v", err))
		return
	}

	valid, apiSession, err := al.validateBearerToken(req.Context(), token, req.URL.Path, req.Method)
	if err != nil {
		if errors.Is(err, ErrTooManyRequests) {
			al.jsonErrorResponse(w, http.StatusTooManyRequests, err)
			return
		}
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if !al.handleBannedIPs(req, valid) {
		return
	}
	if !valid {
		al.bannedUsers.Add(token.AppToken.Username)
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
