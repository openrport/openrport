package middleware

import (
	"log"
	"net/http"
)

type NotFoundRedirectResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *NotFoundRedirectResponseWriter) WriteHeader(status int) {
	w.status = status
	if status != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *NotFoundRedirectResponseWriter) Write(p []byte) (int, error) {
	if w.status != http.StatusNotFound {
		return w.ResponseWriter.Write(p)
	}
	return len(p), nil // lie that it was successfully written
}

func Redirect404(h http.Handler, redirectURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newW := &NotFoundRedirectResponseWriter{ResponseWriter: w}
		h.ServeHTTP(newW, r)
		if newW.status == http.StatusNotFound {
			log.Printf("Redirecting %q to %q.", r.RequestURI, redirectURL)
			http.Redirect(w, r, redirectURL, http.StatusOK)
		}
	}
}
