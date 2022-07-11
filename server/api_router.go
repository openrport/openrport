package chserver

import (
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"

	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/share/security"
)

// define which URL paths vue aka the frontend is using. Listed paths are rewritten to / aka returning index.html
// to make the vue.js history working.
var vueHistoryPaths = []string{
	"dashboard",
	"auth",
	"tunnels",
	"inventory",
	"documentation",
	"commands",
	"scripts",
	"settings",
}

func (al *APIListener) initRouter() {
	r := mux.NewRouter()
	api := r.PathPrefix(allRoutesPrefix).Subrouter()
	api.HandleFunc("/status", al.handleGetStatus).Methods(http.MethodGet)
	api.HandleFunc("/me", al.handleGetMe).Methods(http.MethodGet)
	api.HandleFunc("/me", al.handleChangeMe).Methods(http.MethodPut)
	api.HandleFunc("/me/ip", al.handleGetIP).Methods(http.MethodGet)
	api.HandleFunc("/me/token", al.handlePostToken).Methods(http.MethodPost)
	api.HandleFunc("/me/token", al.handleDeleteToken).Methods(http.MethodDelete)
	api.HandleFunc("/clients", al.handleGetClients).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}", al.wrapClientAccessMiddleware(al.handleGetClient)).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}", al.wrapClientAccessMiddleware(al.handleDeleteClient)).Methods(http.MethodDelete)
	api.HandleFunc("/clients/{client_id}/acl", al.wrapAdminAccessMiddleware(al.handlePostClientACL)).Methods(http.MethodPost)
	api.HandleFunc("/clients/{client_id}/tunnels", al.wrapClientAccessMiddleware(al.handlePutClientTunnel)).Methods(http.MethodPut)
	api.HandleFunc("/clients/{client_id}/tunnels/{tunnel_id}", al.wrapClientAccessMiddleware(al.handleDeleteClientTunnel)).Methods(http.MethodDelete)
	api.HandleFunc("/clients/{client_id}/commands", al.wrapClientAccessMiddleware(al.handlePostCommand)).Methods(http.MethodPost)
	api.HandleFunc("/clients/{client_id}/commands", al.wrapClientAccessMiddleware(al.handleGetCommands)).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}/commands/{job_id}", al.wrapClientAccessMiddleware(al.handleGetCommand)).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}/scripts", al.wrapClientAccessMiddleware(al.handleExecuteScript)).Methods(http.MethodPost)
	api.HandleFunc("/clients/{client_id}/updates-status", al.wrapClientAccessMiddleware(al.handleRefreshUpdatesStatus)).Methods(http.MethodPost)
	api.HandleFunc("/clients/{client_id}/graph-metrics", al.wrapClientAccessMiddleware(al.handleGetClientGraphMetrics)).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}/graph-metrics/{"+routeParamGraphName+"}", al.wrapClientAccessMiddleware(al.handleGetClientGraphMetricsGraph)).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}/metrics", al.wrapClientAccessMiddleware(al.handleGetClientMetrics)).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}/processes", al.wrapClientAccessMiddleware(al.handleGetClientProcesses)).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}/mountpoints", al.wrapClientAccessMiddleware(al.handleGetClientMountpoints)).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}/stored-tunnels", al.wrapClientAccessMiddleware(al.handleGetStoredTunnels)).Methods(http.MethodGet)
	api.HandleFunc("/clients/{client_id}/stored-tunnels", al.wrapClientAccessMiddleware(al.handlePostStoredTunnels)).Methods(http.MethodPost)
	api.HandleFunc("/clients/{client_id}/stored-tunnels/{tunnel_id}", al.wrapClientAccessMiddleware(al.handleDeleteStoredTunnel)).Methods(http.MethodDelete)
	api.HandleFunc("/clients/{client_id}/stored-tunnels/{tunnel_id}", al.wrapClientAccessMiddleware(al.handlePutStoredTunnel)).Methods(http.MethodPut)
	api.HandleFunc("/tunnels", al.handleGetTunnels).Methods(http.MethodGet)
	api.HandleFunc("/client-groups", al.handleGetClientGroups).Methods(http.MethodGet)
	api.HandleFunc("/client-groups", al.wrapAdminAccessMiddleware(al.handlePostClientGroups)).Methods(http.MethodPost)
	api.HandleFunc("/client-groups/{group_id}", al.wrapAdminAccessMiddleware(al.handlePutClientGroup)).Methods(http.MethodPut)
	api.HandleFunc("/client-groups/{group_id}", al.handleGetClientGroup).Methods(http.MethodGet)
	api.HandleFunc("/client-groups/{group_id}", al.wrapAdminAccessMiddleware(al.handleDeleteClientGroup)).Methods(http.MethodDelete)
	api.HandleFunc("/users", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleGetUsers))).Methods(http.MethodGet)
	api.HandleFunc("/users", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleChangeUser))).Methods(http.MethodPost)
	api.HandleFunc("/users/{user_id}", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleChangeUser))).Methods(http.MethodPut)
	api.HandleFunc("/users/{user_id}", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleDeleteUser))).Methods(http.MethodDelete)
	api.HandleFunc("/users/{user_id}/totp-secret", al.wrapStaticPassModeMiddleware(
		al.wrapAdminAccessMiddleware(
			al.wrapTotPEnabledMiddleware(al.handleDeleteUsersTotP),
		),
	)).Methods(http.MethodDelete)
	api.HandleFunc("/user-groups", al.wrapAdminAccessMiddleware(al.handleListUserGroups)).Methods(http.MethodGet)
	// api.HandleFunc("/user-groups/{group_name}", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleGetUserGroup))).Methods(http.MethodGet)
	// api.HandleFunc("/user-groups/{group_name}", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleUpdateUserGroup))).Methods(http.MethodPut)
	// api.HandleFunc("/user-groups/{group_name}", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleDeleteUserGroup))).Methods(http.MethodDelete)
	api.HandleFunc("/commands", al.handlePostMultiClientCommand).Methods(http.MethodPost)
	api.HandleFunc("/commands", al.handleGetMultiClientCommands).Methods(http.MethodGet)
	api.HandleFunc("/commands/{job_id}", al.handleGetMultiClientCommand).Methods(http.MethodGet)
	api.HandleFunc("/commands/{job_id}/jobs", al.handleGetMultiClientCommandJobs).Methods(http.MethodGet)
	api.HandleFunc("/clients-auth", al.wrapAdminAccessMiddleware(al.handleGetClientsAuth)).Methods(http.MethodGet)
	api.HandleFunc("/clients-auth/{client_auth_id}", al.wrapAdminAccessMiddleware(al.handleGetClientAuth)).Methods(http.MethodGet)
	api.HandleFunc("/clients-auth", al.wrapAdminAccessMiddleware(al.handlePostClientsAuth)).Methods(http.MethodPost)
	api.HandleFunc("/clients-auth/{client_auth_id}", al.wrapAdminAccessMiddleware(al.handleDeleteClientAuth)).Methods(http.MethodDelete)
	api.HandleFunc("/vault-admin", al.handleGetVaultStatus).Methods(http.MethodGet)
	api.HandleFunc("/vault-admin/sesame", al.wrapAdminAccessMiddleware(al.handleVaultUnlock)).Methods(http.MethodPost)
	api.HandleFunc("/vault-admin/init", al.wrapAdminAccessMiddleware(al.handleVaultInit)).Methods(http.MethodPost)
	api.HandleFunc("/vault-admin/sesame", al.wrapAdminAccessMiddleware(al.handleVaultLock)).Methods(http.MethodDelete)
	api.HandleFunc("/vault", al.handleListVaultValues).Methods(http.MethodGet)
	api.HandleFunc("/vault", al.handleVaultStoreValue).Methods(http.MethodPost)
	api.HandleFunc("/vault/{"+routeParamVaultValueID+"}", al.handleReadVaultValue).Methods(http.MethodGet)
	api.HandleFunc("/vault/{"+routeParamVaultValueID+"}", al.handleVaultStoreValue).Methods(http.MethodPut)
	api.HandleFunc("/vault/{"+routeParamVaultValueID+"}", al.handleVaultDeleteValue).Methods(http.MethodDelete)
	api.HandleFunc("/library/scripts", al.handleListScripts).Methods(http.MethodGet)
	api.HandleFunc("/library/scripts", al.handleScriptCreate).Methods(http.MethodPost)
	api.HandleFunc("/library/scripts/{"+routeParamScriptValueID+"}", al.handleScriptUpdate).Methods(http.MethodPut)
	api.HandleFunc("/library/scripts/{"+routeParamScriptValueID+"}", al.handleReadScript).Methods(http.MethodGet)
	api.HandleFunc("/library/scripts/{"+routeParamScriptValueID+"}", al.handleDeleteScript).Methods(http.MethodDelete)
	api.HandleFunc("/library/commands", al.handleListCommands).Methods(http.MethodGet)
	api.HandleFunc("/library/commands", al.handleCommandCreate).Methods(http.MethodPost)
	api.HandleFunc("/library/commands/{"+routeParamCommandValueID+"}", al.handleCommandUpdate).Methods(http.MethodPut)
	api.HandleFunc("/library/commands/{"+routeParamCommandValueID+"}", al.handleReadCommand).Methods(http.MethodGet)
	api.HandleFunc("/library/commands/{"+routeParamCommandValueID+"}", al.handleDeleteCommand).Methods(http.MethodDelete)
	api.HandleFunc("/scripts", al.handlePostMultiClientScript).Methods(http.MethodPost)
	api.HandleFunc("/auditlog", al.handleListAuditLog).Methods(http.MethodGet)
	api.HandleFunc("/schedules", al.handleListSchedules).Methods(http.MethodGet)
	api.HandleFunc("/schedules", al.handlePostSchedules).Methods(http.MethodPost)
	api.HandleFunc("/schedules/{schedule_id}", al.handleGetSchedule).Methods(http.MethodGet)
	api.HandleFunc("/schedules/{schedule_id}", al.handleUpdateSchedule).Methods(http.MethodPut)
	api.HandleFunc("/schedules/{schedule_id}", al.handleDeleteSchedule).Methods(http.MethodDelete)
	api.HandleFunc("/files", al.handleFileUploads).Methods(http.MethodPost).Name(filesUploadRouteName)
	api.HandleFunc(totPRoutes, al.wrapTotPEnabledMiddleware(al.handleGetTotP)).Methods(http.MethodGet)
	api.HandleFunc(totPRoutes, al.wrapTotPEnabledMiddleware(al.handlePostTotP)).Methods(http.MethodPost)
	api.HandleFunc(totPRoutes, al.wrapTotPEnabledMiddleware(al.handleDeleteTotP)).Methods(http.MethodDelete)

	// add authorization middleware
	if !al.insecureForTests {
		_ = api.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			route.HandlerFunc(al.wrapWithAuthMiddleware(route.GetHandler(), false))
			return nil
		})
	}

	// all routes defined below do not have authorization middleware, auth is done in each handlers separately
	api.HandleFunc("/login", al.handleGetLogin).Methods(http.MethodGet)
	api.HandleFunc("/login", al.handlePostLogin).Methods(http.MethodPost)
	api.HandleFunc("/logout", al.handleDeleteLogout).Methods(http.MethodDelete)
	api.HandleFunc(verify2FaRoute, al.wrapWithAuthMiddleware(al.handlePostVerify2FAToken(), true)).Methods(http.MethodPost)

	// web sockets
	// common auth middleware is not used due to JS issue https://stackoverflow.com/questions/22383089/is-it-possible-to-use-bearer-authentication-for-websocket-upgrade-requests
	api.HandleFunc("/ws/commands", al.wsAuth(http.HandlerFunc(al.handleCommandsWS))).Methods(http.MethodGet)
	api.HandleFunc("/ws/scripts", al.wsAuth(http.HandlerFunc(al.handleScriptsWS))).Methods(http.MethodGet)
	api.HandleFunc("/ws/uploads", al.wsAuth(http.HandlerFunc(al.handleUploadsWS))).Methods(http.MethodGet)

	if al.config.Server.EnableWsTestEndpoints {
		api.HandleFunc("/test/commands/ui", al.wsCommands)
		api.HandleFunc("/test/scripts/ui", al.wsScripts)
		api.HandleFunc("/test/uploads/ui", al.wsUploads)
	}

	if al.bannedIPs != nil {
		// add middleware to reject banned IPs
		_ = api.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			route.HandlerFunc(security.RejectBannedIPs(route.GetHandler(), al.bannedIPs))
			return nil
		})
	}

	// add max bytes middleware
	_ = api.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		if route.GetName() == filesUploadRouteName {
			route.HandlerFunc(middleware.MaxBytes(route.GetHandler(), al.config.Server.MaxFilePushSize))
		} else {
			route.HandlerFunc(middleware.MaxBytes(route.GetHandler(), al.config.Server.MaxRequestBytes))
		}
		return nil
	})

	docRoot := al.config.API.DocRoot
	if docRoot != "" {
		// Start a http file server with proper Vue.js HTML5 history mode (aka rewrite to /) for the following paths
		r.PathPrefix("/").Handler(middleware.Rewrite404ForVueJs(http.FileServer(http.Dir(docRoot)), vueHistoryPaths))
	}

	if al.requestLogOptions != nil {
		r.Use(func(next http.Handler) http.Handler { return requestlog.WrapWith(next, *al.requestLogOptions) })
	}
	if al.accessLogFile != nil {
		r.Use(func(next http.Handler) http.Handler { return handlers.CombinedLoggingHandler(al.accessLogFile, next) })
	}

	r.Use(handlers.CompressHandler)
	r.Use(handlers.RecoveryHandler(
		handlers.PrintRecoveryStack(true),
		handlers.RecoveryLogger(middleware.NewRecoveryLogger(al.Logger)),
	))

	al.router = r
}
