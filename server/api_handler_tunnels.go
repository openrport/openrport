package chserver

import (
	"net/http"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/clients/clienttunnel"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type TunnelPayload struct {
	models.Remote
	ID       string `json:"id"`
	ClientID string `json:"client_id"`
}

func convertToTunnelPayload(t *clienttunnel.Tunnel, clientID string) TunnelPayload {
	return TunnelPayload{
		Remote:   t.Remote,
		ID:       t.ID,
		ClientID: clientID,
	}
}

func (al *APIListener) handleGetTunnels(w http.ResponseWriter, req *http.Request) {
	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	clients, err := al.clientService.GetUserClients(curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	tunnels := make([]TunnelPayload, 0)
	for _, c := range clients {
		if c.DisconnectedAt != nil {
			continue
		}

		for _, t := range c.Tunnels {
			tunnels = append(tunnels, convertToTunnelPayload(t, c.ID))
		}
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(tunnels))
}
