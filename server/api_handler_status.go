package chserver

import (
	"net/http"

	"github.com/cloudradar-monitoring/rport/server/api"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

func (al *APIListener) handleGetStatus(w http.ResponseWriter, req *http.Request) {
	countActive, err := al.clientService.CountActive()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	countDisconnected, err := al.clientService.CountDisconnected()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	var twoFADelivery string
	if al.twoFASrv.MsgSrv != nil {
		twoFADelivery = al.twoFASrv.MsgSrv.DeliveryMethod()
	} else if al.config.API.TotPEnabled {
		twoFADelivery = "totp_authenticator_app"
	}

	// TODO: (rs): add subdomain stuff here?
	response := api.NewSuccessPayload(map[string]interface{}{
		"version":                   chshare.BuildVersion,
		"clients_connected":         countActive,
		"clients_disconnected":      countDisconnected,
		"fingerprint":               al.fingerprint,
		"connect_url":               al.config.Server.URL,
		"pairing_url":               al.config.Server.PairingURL,
		"tunnel_host":               al.config.Server.TunnelHost,
		"clients_auth_source":       al.clientAuthProvider.Source(),
		"clients_auth_mode":         al.getClientsAuthMode(),
		"users_auth_source":         al.userService.GetProviderType(),
		"group_permissions_enabled": al.userService.SupportsGroupPermissions(),
		"two_fa_enabled":            al.config.API.IsTwoFAOn() || al.config.API.TotPEnabled,
		"two_fa_delivery_method":    twoFADelivery,
		"auditlog":                  al.auditLog.Status(),
		"auth_header":               al.config.API.AuthHeader != "",
		"tunnel_proxy_enabled":      al.config.Server.InternalTunnelProxyConfig.Enabled,
		"excluded_ports":            al.config.Server.ExcludedPortsRaw,
		"used_ports":                al.config.Server.UsedPortsRaw,
	})

	al.writeJSONResponse(w, http.StatusOK, response)
}
