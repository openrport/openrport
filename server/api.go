package chserver

import (
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	routeParamClientID       = "client_id"
	routeParamClientAuthID   = "client_auth_id"
	routeParamUserID         = "user_id"
	routeParamJobID          = "job_id"
	routeParamGroupID        = "group_id"
	routeParamVaultValueID   = "vault_value_id"
	routeParamScriptValueID  = "script_value_id"
	routeParamCommandValueID = "command_value_id"
	routeParamGraphName      = "graph_name"

	ErrCodeMissingRouteVar = "ERR_CODE_MISSING_ROUTE_VAR"
	ErrCodeInvalidRequest  = "ERR_CODE_INVALID_REQUEST"
	ErrCodeAlreadyExist    = "ERR_CODE_ALREADY_EXIST"

	allRoutesPrefix         = "/api/v1"
	authRoutesPrefix        = "/auth"
	authProviderRoute       = "/provider"
	authSettingsRoute       = "/ext/settings"
	authDeviceSettingsRoute = "/ext/settings/device"
	totPRoutes              = "/me/totp-secret"
	verify2FaRoute          = "/verify-2fa"
	filesUploadRouteName    = "files"
)

var apiUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}
