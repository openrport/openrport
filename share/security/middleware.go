package security

import (
	"net/http"

	"github.com/gorilla/mux"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func RejectBannedIPs(bannedIPs *MaxBadAttemptsBanList) mux.MiddlewareFunc {
	return func(f http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := chshare.RemoteIP(r)

			if bannedIPs.IsBanned(ip) {
				http.Error(w, "Too many bad attempts. Please try later.", http.StatusLocked)
				return
			}

			f.ServeHTTP(w, r)
		})
	}
}
