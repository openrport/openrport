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
