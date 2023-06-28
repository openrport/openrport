package chserver

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/realvnc-labs/rport/server/routes"
)

func (al *APIListener) handleGetNotifications(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	// options := query.GetListOptions(request)
	result, err := al.notificationsStorage.List(ctx)
	if err != nil {
		al.jsonError(writer, err)
		return
	}

	al.writeJSONResponse(writer, http.StatusOK, result)
}

func (al *APIListener) handleGetNotificationDetails(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	vars := mux.Vars(request)
	nid := vars[routes.ParamNotificationID]

	notification, found, err := al.notificationsStorage.Details(ctx, nid)
	if err != nil {
		al.jsonError(writer, err)
		return
	}

	if found {
		al.writeJSONResponse(writer, http.StatusOK, notification)
		return
	}

	al.writeJSONResponse(writer, http.StatusNotFound, nil)

}
