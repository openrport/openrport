package security

import (
	"fmt"
	"net"
	"net/http"
)

func RejectBannedIPs(f http.Handler, bannedIPs *MaxBadAttemptsBanList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to split host port for %q: %v", r.RemoteAddr, err), http.StatusInternalServerError)
		}

		if bannedIPs.IsBanned(ip) {
			http.Error(w, "Too many bad attempts. Please try later.", http.StatusLocked)
			return
		}

		f.ServeHTTP(w, r)
	}
}

type ResponseWriterWithStatus struct {
	http.ResponseWriter
	statusCode int
}

func (w *ResponseWriterWithStatus) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func BanIPsOn401(f http.Handler, bannedIPs *MaxBadAttemptsBanList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to split host port for %q: %v", r.RemoteAddr, err), http.StatusInternalServerError)
		}

		if bannedIPs.IsBanned(ip) {
			http.Error(w, "Too many bad attempts. Please try later.", http.StatusLocked)
			return
		}

		newW := &ResponseWriterWithStatus{ResponseWriter: w}
		f.ServeHTTP(newW, r)

		if newW.statusCode == http.StatusUnauthorized {
			bannedIPs.AddBadAttempt(ip)
		}

		// reset bad attempts count only on 2xx since some of APIs could return 4xx before the auth
		if newW.statusCode/100 == 2 {
			bannedIPs.AddSuccessAttempt(ip)
		}
	}
}
