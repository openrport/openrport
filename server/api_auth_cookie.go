package chserver

import (
	"net/http"
	"time"
)

const AuthCookieName = "rport_session"

func (al *APIListener) setAuthCookie(w http.ResponseWriter, req *http.Request, token string, lifetime time.Duration) {
	cookie := &http.Cookie{
		Name:     AuthCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(lifetime.Seconds()),
	}
	http.SetCookie(w, cookie)
}

func (al *APIListener) clearAuthCookie(w http.ResponseWriter, req *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
