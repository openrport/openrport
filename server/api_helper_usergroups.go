package chserver

import (
	"net/http"

	"github.com/cloudradar-monitoring/rport/server/api"
)

func (al *APIListener) handleListUserGroups(w http.ResponseWriter, req *http.Request) {
	items, err := al.userService.ListGroups()
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(items))
}
