package security

import (
	"net/http"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func RejectBannedIPs(f http.Handler, bannedIPs *MaxBadAttemptsBanList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := chshare.RemoteIP(r)

		if bannedIPs.IsBanned(ip) {
			http.Error(w, "Too many bad attempts. Please try later.", http.StatusLocked)
			return
		}

		f.ServeHTTP(w, r)
	}
}
