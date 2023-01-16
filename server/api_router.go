package chserver

import (
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"

	"github.com/cloudradar-monitoring/rport/plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/routes"
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
	api := r.PathPrefix(routes.AllRoutesPrefix).Subrouter()

	secureAPI := api.NewRoute().Subrouter()
	if !al.insecureForTests {
		secureAPI.Use(al.wrapWithAuthMiddleware(false))
	}
	secureAPI.HandleFunc("/status", al.handleGetStatus).Methods(http.MethodGet)
	secureAPI.HandleFunc("/me", al.handleGetMe).Methods(http.MethodGet)
	secureAPI.HandleFunc("/me", al.handleChangeMe).Methods(http.MethodPut)
	secureAPI.HandleFunc("/me/ip", al.handleGetIP).Methods(http.MethodGet)
	secureAPI.HandleFunc("/me/token", al.handlePostToken).Methods(http.MethodPost)
	secureAPI.HandleFunc("/me/token", al.handleDeleteToken).Methods(http.MethodDelete)

	secureAPI.HandleFunc("/clients", al.handleGetClients).Methods(http.MethodGet)
	clientDetails := secureAPI.PathPrefix("/clients/{client_id}").Subrouter()
	clientDetails.Use(al.wrapClientAccessMiddleware)
	clientDetails.HandleFunc("", al.handleGetClient).Methods(http.MethodGet)
	clientDetails.HandleFunc("", al.handleDeleteClient).Methods(http.MethodDelete)
	clientDetails.Handle("/acl", al.wrapAdminAccessMiddleware(http.HandlerFunc(al.handlePostClientACL))).Methods(http.MethodPost)
	clientDetails.Handle("/scripts", al.permissionsMiddleware(users.PermissionScripts)(http.HandlerFunc(al.handleExecuteScript))).Methods(http.MethodPost)

	clientCommands := clientDetails.PathPrefix("/commands").Subrouter()
	clientCommands.Use(al.permissionsMiddleware(users.PermissionCommands))
	clientCommands.HandleFunc("", al.handlePostCommand).Methods(http.MethodPost)
	clientCommands.HandleFunc("", al.handleGetCommands).Methods(http.MethodGet)
	clientCommands.HandleFunc("/{job_id}", al.handleGetCommand).Methods(http.MethodGet)

	clientTunnels := clientDetails.NewRoute().Subrouter()
	clientTunnels.Use(al.permissionsMiddleware(users.PermissionTunnels))
	clientTunnels.HandleFunc("/tunnels", al.handlePutClientTunnel).Methods(http.MethodPut)
	clientTunnels.HandleFunc("/tunnels/{tunnel_id}", al.handleDeleteClientTunnel).Methods(http.MethodDelete)
	clientTunnels.HandleFunc("/stored-tunnels", al.handleGetStoredTunnels).Methods(http.MethodGet)
	clientTunnels.HandleFunc("/stored-tunnels", al.handlePostStoredTunnels).Methods(http.MethodPost)
	clientTunnels.HandleFunc("/stored-tunnels/{tunnel_id}", al.handleDeleteStoredTunnel).Methods(http.MethodDelete)
	clientTunnels.HandleFunc("/stored-tunnels/{tunnel_id}", al.handlePutStoredTunnel).Methods(http.MethodPut)

	clientMonitoring := clientDetails.NewRoute().Subrouter()
	clientMonitoring.Use(al.permissionsMiddleware(users.PermissionMonitoring))
	clientMonitoring.HandleFunc("/updates-status", al.handleRefreshUpdatesStatus).Methods(http.MethodPost)
	if al.Server.config.Monitoring.Enabled {
		clientMonitoring.HandleFunc("/graph-metrics", al.handleGetClientGraphMetrics).Methods(http.MethodGet)
		clientMonitoring.HandleFunc("/graph-metrics/{"+routes.ParamGraphName+"}", al.handleGetClientGraphMetricsGraph).Methods(http.MethodGet)
		clientMonitoring.HandleFunc("/metrics", al.handleGetClientMetrics).Methods(http.MethodGet)
		clientMonitoring.HandleFunc("/processes", al.handleGetClientProcesses).Methods(http.MethodGet)
		clientMonitoring.HandleFunc("/mountpoints", al.handleGetClientMountpoints).Methods(http.MethodGet)
	} else {
		clientMonitoring.HandleFunc("/graph-metrics", al.handleMonitoringDisabled).Methods(http.MethodGet)
		clientMonitoring.HandleFunc("/graph-metrics/{"+routes.ParamGraphName+"}", al.handleMonitoringDisabled).Methods(http.MethodGet)
		clientMonitoring.HandleFunc("/metrics", al.handleMonitoringDisabled).Methods(http.MethodGet)
		clientMonitoring.HandleFunc("/processes", al.handleMonitoringDisabled).Methods(http.MethodGet)
		clientMonitoring.HandleFunc("/mountpoints", al.handleMonitoringDisabled).Methods(http.MethodGet)
	}

	secureAPI.Handle("/tunnels", al.permissionsMiddleware(users.PermissionTunnels)(http.HandlerFunc(al.handleGetTunnels))).Methods(http.MethodGet)
	secureAPI.Handle("/auditlog", al.permissionsMiddleware(users.PermissionsAuditLog)(http.HandlerFunc(al.handleListAuditLog))).Methods(http.MethodGet)
	secureAPI.Handle("/files", al.permissionsMiddleware(users.PermissionUploads)(http.HandlerFunc(al.handleFileUploads))).Methods(http.MethodPost).Name(routes.FilesUploadRouteName)

	secureAPI.HandleFunc("/client-groups", al.handleGetClientGroups).Methods(http.MethodGet)
	secureAPI.HandleFunc("/client-groups/{group_id}", al.handleGetClientGroup).Methods(http.MethodGet)

	adminOnly := secureAPI.NewRoute().Subrouter()
	adminOnly.Use(al.wrapAdminAccessMiddleware)
	adminOnly.HandleFunc("/client-groups", al.handlePostClientGroups).Methods(http.MethodPost)
	adminOnly.HandleFunc("/client-groups/{group_id}", al.handlePutClientGroup).Methods(http.MethodPut)
	adminOnly.HandleFunc("/client-groups/{group_id}", al.handleDeleteClientGroup).Methods(http.MethodDelete)
	adminOnly.HandleFunc("/users", al.wrapStaticPassModeMiddleware(al.handleGetUsers)).Methods(http.MethodGet)
	adminOnly.HandleFunc("/users", al.wrapStaticPassModeMiddleware(al.handleChangeUser)).Methods(http.MethodPost)
	adminOnly.HandleFunc("/users/{user_id}", al.wrapStaticPassModeMiddleware(al.handleChangeUser)).Methods(http.MethodPut)
	adminOnly.HandleFunc("/users/{user_id}", al.wrapStaticPassModeMiddleware(al.handleDeleteUser)).Methods(http.MethodDelete)
	adminOnly.HandleFunc("/users/{user_id}/totp-secret", al.wrapStaticPassModeMiddleware(
		al.wrapTotPEnabledMiddleware(al.handleDeleteUsersTotP),
	)).Methods(http.MethodDelete)

	adminOnly.HandleFunc("/users/{user_id}/sessions", al.handleGetUserAPISessions).Methods(http.MethodGet)
	adminOnly.HandleFunc("/users/{user_id}/sessions", al.handleDeleteAllUserAPISessions).Methods(http.MethodDelete)
	adminOnly.HandleFunc("/users/{user_id}/sessions/{session_id}", al.handleDeleteUserAPISession).Methods(http.MethodDelete)

	adminOnly.HandleFunc("/user-groups", al.handleListUserGroups).Methods(http.MethodGet)
	adminOnly.HandleFunc("/user-groups/{group_name}", al.wrapStaticPassModeMiddleware(al.handleGetUserGroup)).Methods(http.MethodGet)
	adminOnly.HandleFunc("/user-groups/{group_name}", al.wrapStaticPassModeMiddleware(al.handleUpdateUserGroup)).Methods(http.MethodPut)
	adminOnly.HandleFunc("/user-groups/{group_name}", al.wrapStaticPassModeMiddleware(al.handleDeleteUserGroup)).Methods(http.MethodDelete)
	adminOnly.HandleFunc("/clients-auth", al.handleGetClientsAuth).Methods(http.MethodGet)
	adminOnly.HandleFunc("/clients-auth/{client_auth_id}", al.handleGetClientAuth).Methods(http.MethodGet)
	adminOnly.HandleFunc("/clients-auth", al.handlePostClientsAuth).Methods(http.MethodPost)
	adminOnly.HandleFunc("/clients-auth/{client_auth_id}", al.handleDeleteClientAuth).Methods(http.MethodDelete)

	commands := secureAPI.NewRoute().Subrouter()
	commands.Use(al.permissionsMiddleware(users.PermissionCommands))
	commands.HandleFunc("/commands", al.handlePostMultiClientCommand).Methods(http.MethodPost)
	commands.HandleFunc("/commands", al.handleGetMultiClientCommands).Methods(http.MethodGet)
	commands.HandleFunc("/commands/{job_id}", al.handleGetMultiClientCommand).Methods(http.MethodGet)
	commands.HandleFunc("/commands/{job_id}/jobs", al.handleGetMultiClientCommandJobs).Methods(http.MethodGet)
	commands.HandleFunc("/library/commands", al.handleListCommands).Methods(http.MethodGet)
	commands.HandleFunc("/library/commands", al.handleCommandCreate).Methods(http.MethodPost)
	commands.HandleFunc("/library/commands/{"+routes.ParamCommandValueID+"}", al.handleCommandUpdate).Methods(http.MethodPut)
	commands.HandleFunc("/library/commands/{"+routes.ParamCommandValueID+"}", al.handleReadCommand).Methods(http.MethodGet)
	commands.HandleFunc("/library/commands/{"+routes.ParamCommandValueID+"}", al.handleDeleteCommand).Methods(http.MethodDelete)

	scripts := secureAPI.NewRoute().Subrouter()
	scripts.Use(al.permissionsMiddleware(users.PermissionScripts))
	scripts.HandleFunc("/library/scripts", al.handleListScripts).Methods(http.MethodGet)
	scripts.HandleFunc("/library/scripts", al.handleScriptCreate).Methods(http.MethodPost)
	scripts.HandleFunc("/library/scripts/{"+routes.ParamScriptValueID+"}", al.handleScriptUpdate).Methods(http.MethodPut)
	scripts.HandleFunc("/library/scripts/{"+routes.ParamScriptValueID+"}", al.handleReadScript).Methods(http.MethodGet)
	scripts.HandleFunc("/library/scripts/{"+routes.ParamScriptValueID+"}", al.handleDeleteScript).Methods(http.MethodDelete)
	scripts.HandleFunc("/scripts", al.handlePostMultiClientScript).Methods(http.MethodPost)

	vault := secureAPI.NewRoute().Subrouter()
	vault.Use(al.permissionsMiddleware(users.PermissionVault))
	vault.HandleFunc("/vault-admin", al.handleGetVaultStatus).Methods(http.MethodGet)
	vault.Handle("/vault-admin/sesame", al.wrapAdminAccessMiddleware(http.HandlerFunc(al.handleVaultUnlock))).Methods(http.MethodPost)
	vault.Handle("/vault-admin/init", al.wrapAdminAccessMiddleware(http.HandlerFunc(al.handleVaultInit))).Methods(http.MethodPost)
	vault.Handle("/vault-admin/sesame", al.wrapAdminAccessMiddleware(http.HandlerFunc(al.handleVaultLock))).Methods(http.MethodDelete)
	vault.HandleFunc("/vault", al.handleListVaultValues).Methods(http.MethodGet)
	vault.HandleFunc("/vault", al.handleVaultStoreValue).Methods(http.MethodPost)
	vault.HandleFunc("/vault/{"+routes.ParamVaultValueID+"}", al.handleReadVaultValue).Methods(http.MethodGet)
	vault.HandleFunc("/vault/{"+routes.ParamVaultValueID+"}", al.handleVaultStoreValue).Methods(http.MethodPut)
	vault.HandleFunc("/vault/{"+routes.ParamVaultValueID+"}", al.handleVaultDeleteValue).Methods(http.MethodDelete)

	schedules := secureAPI.PathPrefix("/schedules").Subrouter()
	schedules.Use(al.permissionsMiddleware(users.PermissionScheduler))
	schedules.HandleFunc("", al.handleListSchedules).Methods(http.MethodGet)
	schedules.HandleFunc("", al.handlePostSchedules).Methods(http.MethodPost)
	schedules.HandleFunc("/{schedule_id}", al.handleGetSchedule).Methods(http.MethodGet)
	schedules.HandleFunc("/{schedule_id}", al.handleUpdateSchedule).Methods(http.MethodPut)
	schedules.HandleFunc("/{schedule_id}", al.handleDeleteSchedule).Methods(http.MethodDelete)

	secureAPI.HandleFunc(routes.TotPRoutes, al.wrapTotPEnabledMiddleware(al.handleGetTotP)).Methods(http.MethodGet)
	secureAPI.HandleFunc(routes.TotPRoutes, al.wrapTotPEnabledMiddleware(al.handlePostTotP)).Methods(http.MethodPost)
	secureAPI.HandleFunc(routes.TotPRoutes, al.wrapTotPEnabledMiddleware(al.handleDeleteTotP)).Methods(http.MethodDelete)

	// all routes defined below do not have authorization middleware, auth is done in each handler separately
	api.HandleFunc("/login", al.handleGetLogin).Methods(http.MethodGet)
	api.HandleFunc("/login", al.handlePostLogin).Methods(http.MethodPost)
	api.HandleFunc("/logout", al.handleDeleteLogout).Methods(http.MethodDelete)
	api.Handle(routes.Verify2FaRoute, al.wrapWithAuthMiddleware(true)(al.handlePostVerify2FAToken())).Methods(http.MethodPost)

	// web sockets
	// common auth middleware is not used due to JS issue https://stackoverflow.com/questions/22383089/is-it-possible-to-use-bearer-authentication-for-websocket-upgrade-requests
	api.HandleFunc("/ws/commands", al.wsAuth(al.permissionsMiddleware(users.PermissionCommands)(http.HandlerFunc(al.handleCommandsWS)))).Methods(http.MethodGet)
	api.HandleFunc("/ws/scripts", al.wsAuth(al.permissionsMiddleware(users.PermissionScripts)(http.HandlerFunc(al.handleScriptsWS)))).Methods(http.MethodGet)
	api.HandleFunc("/ws/uploads", al.wsAuth(al.permissionsMiddleware(users.PermissionUploads)(http.HandlerFunc(al.handleUploadsWS)))).Methods(http.MethodGet)

	if al.config.Server.EnableWsTestEndpoints {
		api.HandleFunc("/test/commands/ui", al.wsCommands)
		api.HandleFunc("/test/scripts/ui", al.wsScripts)
		api.HandleFunc("/test/uploads/ui", al.wsUploads)
	}

	if al.bannedIPs != nil {
		api.Use(security.RejectBannedIPs(al.bannedIPs))
	}

	// add max bytes middleware
	_ = api.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		if route.GetName() == routes.FilesUploadRouteName {
			route.HandlerFunc(middleware.MaxBytes(route.GetHandler(), al.config.Server.MaxFilePushSize))
		} else {
			route.HandlerFunc(middleware.MaxBytes(route.GetHandler(), al.config.Server.MaxRequestBytes))
		}
		return nil
	})

	plusRouter := api.PathPrefix("/plus").Subrouter()
	plusRouter.HandleFunc("/status", al.handlePlusStatus).Methods(http.MethodGet)

	authRouter := api.PathPrefix(routes.AuthRoutesPrefix).Subrouter()
	authRouter.HandleFunc(routes.AuthProviderRoute, al.handleGetAuthProvider).Methods(http.MethodGet)
	authRouter.HandleFunc(routes.AuthSettingsRoute, al.handleGetAuthSettings).Methods(http.MethodGet)
	authRouter.HandleFunc(routes.AuthDeviceSettingsRoute, al.handleGetAuthDeviceSettings).Methods(http.MethodGet)

	if al.config.PlusOAuthEnabled() {
		api.HandleFunc(oauth.DefaultLoginURI, al.handleOAuthAuthorizationCode).Methods(http.MethodGet)
		api.HandleFunc(oauth.DefaultDeviceLoginURI, al.handleGetDeviceAuth).Methods(http.MethodGet)
	}

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
