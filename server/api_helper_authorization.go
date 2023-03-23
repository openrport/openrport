package chserver

import (
	"net/http"

	chshare "github.com/realvnc-labs/rport/share"
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
