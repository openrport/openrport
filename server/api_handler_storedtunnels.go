package chserver

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/clients/storedtunnels"
	"github.com/realvnc-labs/rport/server/routes"
	"github.com/realvnc-labs/rport/share/query"
)

func (al *APIListener) handleGetStoredTunnels(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]

	client, err := al.clientService.GetByID(clientID)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %q not found", clientID))
		return
	}

	options := query.GetListOptions(req)
	result, err := al.storedTunnels.List(ctx, options, client.GetID())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, result)
}

func (al *APIListener) handlePostStoredTunnels(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]

	client, err := al.clientService.GetByID(clientID)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %q not found", clientID))
		return
	}

	storedTunnel := &storedtunnels.StoredTunnel{}
	err = parseRequestBody(req.Body, storedTunnel)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	result, err := al.storedTunnels.Create(ctx, client.GetID(), storedTunnel)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(result))
}

func (al *APIListener) handleDeleteStoredTunnel(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]
	tunnelID := vars["tunnel_id"]

	client, err := al.clientService.GetByID(clientID)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %q not found", clientID))
		return
	}

	// ED TODO: need to check if its the same user that owned this, but need to fetch the tunnel first
	err = al.storedTunnels.Delete(ctx, client.GetID(), tunnelID)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) handlePutStoredTunnel(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]
	tunnelID := vars["tunnel_id"]

	client, err := al.clientService.GetByID(clientID)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %q not found", clientID))
		return
	}

	storedTunnel := &storedtunnels.StoredTunnel{}
	err = parseRequestBody(req.Body, storedTunnel)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	storedTunnel.ID = tunnelID

	result, err := al.storedTunnels.Update(ctx, client.GetID(), storedTunnel)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(result))
}
