package chserver

import (
	"net/http"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func (al *APIListener) handleBannedIPs(r *http.Request, authorized bool) (ok bool) {
	if al.bannedIPs != nil {
		ip := chshare.RemoteIP(r)

		if authorized {
			al.bannedIPs.AddSuccessAttempt(ip)
		} else {
			al.bannedIPs.AddBadAttempt(ip)
		}
	}

	return true
}
