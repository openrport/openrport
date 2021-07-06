package chserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/tomasen/realip"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/api"
	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/clientsauth"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/script"
	"github.com/cloudradar-monitoring/rport/server/vault"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/enums"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
	"github.com/cloudradar-monitoring/rport/share/security"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

const (
	queryParamSort = "sort"

	routeParamClientID      = "client_id"
	routeParamUserID        = "user_id"
	routeParamJobID         = "job_id"
	routeParamGroupID       = "group_id"
	routeParamVaultValueID  = "vault_value_id"
	routeParamScriptValueID = "script_value_id"

	isPowershellScriptParam = "isPowershell"
	isSudoScriptParam       = "isSudo"
	cwdScriptParam          = "cwd"
	timeoutScriptParam      = "timeout"

	ErrCodeMissingRouteVar = "ERR_CODE_MISSING_ROUTE_VAR"
	ErrCodeInvalidRequest  = "ERR_CODE_INVALID_REQUEST"
	ErrCodeAlreadyExist    = "ERR_CODE_ALREADY_EXIST"

	minVersionScriptExecSupport = "0.1.35"
)

var validInputShell = []string{"cmd", "powershell"}

var generateNewJobID = func() string {
	return random.UUID4()
}

var apiUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type JobProvider interface {
	GetByJID(clientID, jid string) (*models.Job, error)
	GetSummariesByClientID(clientID string) ([]*models.JobSummary, error)
	GetByMultiJobID(jid string) ([]*models.Job, error)
	// SaveJob creates or updates a job
	SaveJob(job *models.Job) error
	// CreateJob creates a new job. If already exist with a given JID - do nothing and return nil
	CreateJob(job *models.Job) error
	GetMultiJob(jid string) (*models.MultiJob, error)
	GetAllMultiJobSummaries() ([]*models.MultiJobSummary, error)
	SaveMultiJob(multiJob *models.MultiJob) error
	Close() error
}

func (al *APIListener) wrapWithAuthMiddleware(f http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorized, username, err := al.lookupUser(r)
		if err != nil {
			if errors.Is(err, ErrTooManyRequests) {
				al.jsonErrorResponse(w, http.StatusTooManyRequests, err)
				return
			}
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if !al.handleBannedIPs(w, r, authorized) {
			return
		}

		if !authorized || username == "" {
			al.bannedUsers.Add(username)
			al.jsonErrorResponse(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}

		newCtx := api.WithUser(r.Context(), username)
		f.ServeHTTP(w, r.WithContext(newCtx))
	}
}

func (al *APIListener) handleBannedIPs(w http.ResponseWriter, r *http.Request, authorized bool) (ok bool) {
	if al.bannedIPs != nil {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("failed to split host port for %q: %v", r.RemoteAddr, err))
			return false
		}

		if authorized {
			al.bannedIPs.AddSuccessAttempt(ip)
		} else {
			al.bannedIPs.AddBadAttempt(ip)
		}
	}

	return true
}

func (al *APIListener) initRouter() {
	r := mux.NewRouter()
	sub := r.PathPrefix("/api/v1").Subrouter()
	sub.HandleFunc("/status", al.handleGetStatus).Methods(http.MethodGet)
	sub.HandleFunc("/me", al.handleGetMe).Methods(http.MethodGet)
	sub.HandleFunc("/me", al.handleChangeMe).Methods(http.MethodPut)
	sub.HandleFunc("/me/ip", al.handleGetIP).Methods(http.MethodGet)
	sub.HandleFunc("/clients", al.handleGetClients).Methods(http.MethodGet)
	sub.HandleFunc("/clients/{client_id}", al.handleDeleteClient).Methods(http.MethodDelete)
	sub.HandleFunc("/clients/{client_id}/tunnels", al.handlePutClientTunnel).Methods(http.MethodPut)
	sub.HandleFunc("/clients/{client_id}/tunnels/{tunnel_id}", al.handleDeleteClientTunnel).Methods(http.MethodDelete)
	sub.HandleFunc("/clients/{client_id}/commands", al.handlePostCommand).Methods(http.MethodPost)
	sub.HandleFunc("/clients/{client_id}/commands", al.handleGetCommands).Methods(http.MethodGet)
	sub.HandleFunc("/clients/{client_id}/commands/{job_id}", al.handleGetCommand).Methods(http.MethodGet)
	sub.HandleFunc("/clients/{client_id}/scripts", al.handleExecuteScript).Methods(http.MethodPost)
	sub.HandleFunc("/client-groups", al.handleGetClientGroups).Methods(http.MethodGet)
	sub.HandleFunc("/client-groups", al.handlePostClientGroups).Methods(http.MethodPost)
	sub.HandleFunc("/client-groups/{group_id}", al.handlePutClientGroup).Methods(http.MethodPut)
	sub.HandleFunc("/client-groups/{group_id}", al.handleGetClientGroup).Methods(http.MethodGet)
	sub.HandleFunc("/client-groups/{group_id}", al.handleDeleteClientGroup).Methods(http.MethodDelete)
	sub.HandleFunc("/users", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleGetUsers))).Methods(http.MethodGet)
	sub.HandleFunc("/users", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleChangeUser))).Methods(http.MethodPost)
	sub.HandleFunc("/users/{user_id}", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleChangeUser))).Methods(http.MethodPut)
	sub.HandleFunc("/users/{user_id}", al.wrapStaticPassModeMiddleware(al.wrapAdminAccessMiddleware(al.handleDeleteUser))).Methods(http.MethodDelete)
	sub.HandleFunc("/commands", al.handlePostMultiClientCommand).Methods(http.MethodPost)
	sub.HandleFunc("/commands", al.handleGetMultiClientCommands).Methods(http.MethodGet)
	sub.HandleFunc("/commands/{job_id}", al.handleGetMultiClientCommand).Methods(http.MethodGet)
	sub.HandleFunc("/clients-auth", al.handleGetClientsAuth).Methods(http.MethodGet)
	sub.HandleFunc("/clients-auth", al.handlePostClientsAuth).Methods(http.MethodPost)
	sub.HandleFunc("/clients-auth/{client_auth_id}", al.handleDeleteClientAuth).Methods(http.MethodDelete)
	sub.HandleFunc("/vault-admin", al.handleGetVaultStatus).Methods(http.MethodGet)
	sub.HandleFunc("/vault-admin/sesame", al.wrapAdminAccessMiddleware(al.handleVaultUnlock)).Methods(http.MethodPost)
	sub.HandleFunc("/vault-admin/init", al.wrapAdminAccessMiddleware(al.handleVaultInit)).Methods(http.MethodPost)
	sub.HandleFunc("/vault-admin/sesame", al.wrapAdminAccessMiddleware(al.handleVaultLock)).Methods(http.MethodDelete)
	sub.HandleFunc("/vault", al.handleListVaultValues).Methods(http.MethodGet)
	sub.HandleFunc("/vault", al.handleVaultStoreValue).Methods(http.MethodPost)
	sub.HandleFunc("/vault/{"+routeParamVaultValueID+"}", al.handleReadVaultValue).Methods(http.MethodGet)
	sub.HandleFunc("/vault/{"+routeParamVaultValueID+"}", al.handleVaultStoreValue).Methods(http.MethodPut)
	sub.HandleFunc("/vault/{"+routeParamVaultValueID+"}", al.handleVaultDeleteValue).Methods(http.MethodDelete)
	sub.HandleFunc("/library/scripts", al.handleListScripts).Methods(http.MethodGet)
	sub.HandleFunc("/library/scripts", al.handleScriptCreate).Methods(http.MethodPost)
	sub.HandleFunc("/library/scripts/{"+routeParamScriptValueID+"}", al.handleScriptUpdate).Methods(http.MethodPut)
	sub.HandleFunc("/library/scripts/{"+routeParamScriptValueID+"}", al.handleReadScript).Methods(http.MethodGet)
	sub.HandleFunc("/library/scripts/{"+routeParamScriptValueID+"}", al.handleDeleteScript).Methods(http.MethodDelete)

	// add authorization middleware
	if !al.insecureForTests {
		_ = sub.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			route.HandlerFunc(al.wrapWithAuthMiddleware(route.GetHandler()))
			return nil
		})
	}

	// all routes defined below do not have authorization middleware, auth is done in each handlers separately
	sub.HandleFunc("/login", al.handleGetLogin).Methods(http.MethodGet)
	sub.HandleFunc("/login", al.handlePostLogin).Methods(http.MethodPost)
	sub.HandleFunc("/logout", al.handleDeleteLogout).Methods(http.MethodDelete)
	sub.HandleFunc("/verify-2fa", al.handlePostVerify2FAToken).Methods(http.MethodPost)

	// web sockets
	// common auth middleware is not used due to JS issue https://stackoverflow.com/questions/22383089/is-it-possible-to-use-bearer-authentication-for-websocket-upgrade-requests
	sub.HandleFunc("/ws/commands", al.wsAuth(http.HandlerFunc(al.handleCommandsWS))).Methods(http.MethodGet)
	sub.HandleFunc("/ws/scripts", al.wsAuth(http.HandlerFunc(al.handleScriptsWS))).Methods(http.MethodGet)

	if al.config.Server.EnableWsTestEndpoints {
		sub.HandleFunc("/test/commands/ui", al.wsCommands)
		sub.HandleFunc("/test/scripts/ui", al.wsScripts)
	}

	if al.bannedIPs != nil {
		// add middleware to reject banned IPs
		_ = sub.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			route.HandlerFunc(security.RejectBannedIPs(route.GetHandler(), al.bannedIPs))
			return nil
		})
	}

	// add max bytes middleware
	_ = sub.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		route.HandlerFunc(middleware.MaxBytes(route.GetHandler(), al.config.Server.MaxRequestBytes))
		return nil
	})

	al.router = r
}

func (al *APIListener) writeJSONResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	b, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	if _, err := w.Write(b); err != nil {
		al.Errorf("error writing response: %s", err)
	}
}

func (al *APIListener) jsonErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	al.writeJSONResponse(w, statusCode, api.NewErrorPayload(err))
}

func (al *APIListener) jsonError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	errCode := strconv.Itoa(statusCode)
	msg := err.Error()
	var apiErr errors2.APIError
	var apiErrs errors2.APIErrors
	switch {
	case errors.As(err, &apiErr):
		statusCode = apiErr.Code
		errCode = strconv.Itoa(statusCode)
		al.writeJSONResponse(w, statusCode, api.NewErrorPayloadWithCode(errCode, msg, ""))
		return
	case errors.As(err, &apiErrs):
		if len(apiErrs) > 0 {
			statusCode = apiErrs[0].Code
		}
		al.writeJSONResponse(w, statusCode, api.NewAPIErrorsPayloadWithCode(apiErrs))
		return
	}

	al.writeJSONResponse(w, statusCode, api.NewErrorPayloadWithCode(errCode, msg, ""))
}

func (al *APIListener) jsonErrorResponseWithErrCode(w http.ResponseWriter, statusCode int, errCode, title string) {
	al.writeJSONResponse(w, statusCode, api.NewErrorPayloadWithCode(errCode, title, ""))
}

func (al *APIListener) jsonErrorResponseWithTitle(w http.ResponseWriter, statusCode int, title string) {
	al.writeJSONResponse(w, statusCode, api.NewErrorPayloadWithCode("", title, ""))
}

func (al *APIListener) jsonErrorResponseWithDetail(w http.ResponseWriter, statusCode int, errCode, title, detail string) {
	al.writeJSONResponse(w, statusCode, api.NewErrorPayloadWithCode(errCode, title, detail))
}

func (al *APIListener) jsonErrorResponseWithError(w http.ResponseWriter, statusCode int, errCode, title string, err error) {
	var detail string
	if err != nil {
		detail = err.Error()
	}
	al.writeJSONResponse(w, statusCode, api.NewErrorPayloadWithCode(errCode, title, detail))
}

type twoFAResponse struct {
	SendTo         string `json:"send_to"`
	DeliveryMethod string `json:"delivery_method"`
}

type loginResponse struct {
	Token *string        `json:"token"`  // null if 2fa is on
	TwoFA *twoFAResponse `json:"two_fa"` // null if 2fa is off
}

func (al *APIListener) handleGetLogin(w http.ResponseWriter, req *http.Request) {
	basicUser, basicPwd, basicAuthProvided := req.BasicAuth()
	if !basicAuthProvided {
		// TODO: consider to move this check from all API endpoints to middleware similar to https://github.com/cloudradar-monitoring/rport/pull/199/commits/4ca1ca9f56c557762d79a60ffc96d2de47f3133c
		// ban IP if it sends a lot of bad requests
		if !al.handleBannedIPs(w, req, false) {
			return
		}
		al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, "basic auth is required")
		return
	}

	al.handleLogin(basicUser, basicPwd, w, req)
}

func (al *APIListener) handleLogin(username, pwd string, w http.ResponseWriter, req *http.Request) {
	if al.bannedUsers.IsBanned(username) {
		al.jsonErrorResponseWithTitle(w, http.StatusTooManyRequests, ErrTooManyRequests.Error())
		return
	}

	if username == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, "username is required")
		return
	}

	authorized, err := al.validateCredentials(username, pwd)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	if !al.handleBannedIPs(w, req, authorized) {
		return
	}

	if !authorized {
		al.bannedUsers.Add(username)
		al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if al.config.API.IsTwoFAOn() {
		sendTo, err := al.twoFASrv.SendToken(req.Context(), username)
		if err != nil {
			al.jsonError(w, err)
			return
		}

		al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(loginResponse{
			TwoFA: &twoFAResponse{
				SendTo:         sendTo,
				DeliveryMethod: al.twoFASrv.MsgSrv.DeliveryMethod(),
			},
		}))
		return
	}

	al.sendJWTToken(username, w, req)
}

func (al *APIListener) sendJWTToken(username string, w http.ResponseWriter, req *http.Request) {
	lifetime, err := parseTokenLifetime(req)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	tokenStr, err := al.createAuthToken(lifetime, username)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := api.NewSuccessPayload(loginResponse{
		Token: &tokenStr,
	})
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handlePostLogin(w http.ResponseWriter, req *http.Request) {
	username, pwd, err := parseLoginPostRequestBody(req)
	if err != nil {
		// ban IP if it sends a lot of bad requests
		if !al.handleBannedIPs(w, req, false) {
			return
		}
		al.jsonError(w, err)
		return
	}

	al.handleLogin(username, pwd, w, req)
}

func parseLoginPostRequestBody(req *http.Request) (string, string, error) {
	reqContentType := req.Header.Get("Content-Type")
	if reqContentType == "application/x-www-form-urlencoded" {
		err := req.ParseForm()
		if err != nil {
			return "", "", errors2.APIError{
				Err:  fmt.Errorf("failed to parse form: %v", err),
				Code: http.StatusBadRequest,
			}
		}
		return req.PostForm.Get("username"), req.PostForm.Get("password"), nil
	}
	if reqContentType == "application/json" {
		type loginReq struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var params loginReq
		err := json.NewDecoder(req.Body).Decode(&params)
		if err != nil {
			return "", "", errors2.APIError{
				Err:  fmt.Errorf("failed to parse request body: %v", err),
				Code: http.StatusBadRequest,
			}
		}
		return params.Username, params.Password, nil
	}
	return "", "", errors2.APIError{
		Message: fmt.Sprintf("unsupported content type: %s", reqContentType),
		Code:    http.StatusBadRequest,
	}
}

func parseTokenLifetime(req *http.Request) (time.Duration, error) {
	lifetimeStr := req.URL.Query().Get("token-lifetime")
	if lifetimeStr == "" {
		lifetimeStr = "0"
	}
	lifetime, err := strconv.ParseInt(lifetimeStr, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("invalid token-lifetime : %s", err)
	}
	result := time.Duration(lifetime) * time.Second
	if result > maxTokenLifetime {
		return 0, fmt.Errorf("requested token lifetime exceeds max allowed %d", maxTokenLifetime/time.Second)
	}
	if result <= 0 {
		result = defaultTokenLifetime
	}
	return result, nil
}

func (al *APIListener) handleDeleteLogout(w http.ResponseWriter, req *http.Request) {
	token, tokenProvided := getBearerToken(req)
	if token == "" || !tokenProvided {
		// ban IP if it sends a lot of bad requests
		if !al.handleBannedIPs(w, req, false) {
			return
		}
		al.jsonErrorResponse(w, http.StatusBadRequest, fmt.Errorf("authorization Bearer token required"))
		return
	}

	valid, user, apiSession, err := al.validateBearerToken(token)
	if err != nil {
		if errors.Is(err, ErrTooManyRequests) {
			al.jsonErrorResponse(w, http.StatusTooManyRequests, err)
			return
		}
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if !al.handleBannedIPs(w, req, valid) {
		return
	}
	if !valid {
		al.bannedUsers.Add(user)
		al.jsonErrorResponse(w, http.StatusBadRequest, fmt.Errorf("token is invalid or expired"))
		return
	}

	err = al.apiSessionRepo.Delete(apiSession)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) handlePostVerify2FAToken(w http.ResponseWriter, req *http.Request) {
	username, err := al.parseAndValidate2FATokenRequest(req)
	if err != nil {
		if !al.handleBannedIPs(w, req, false) {
			return
		}
		al.jsonError(w, err)
		return
	}

	al.sendJWTToken(username, w, req)
}

func (al *APIListener) parseAndValidate2FATokenRequest(req *http.Request) (username string, err error) {
	if !al.config.API.IsTwoFAOn() {
		return "", errors2.APIError{
			Code:    http.StatusConflict,
			Message: "2fa is disabled",
		}
	}

	var reqBody struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	}
	err = json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		return "", errors2.APIError{
			Code: http.StatusBadRequest,
			Err:  fmt.Errorf("failed to parse request body: %v", err),
		}
	}

	if al.bannedUsers.IsBanned(reqBody.Username) {
		return reqBody.Username, errors2.APIError{
			Code: http.StatusTooManyRequests,
			Err:  ErrTooManyRequests,
		}
	}

	if reqBody.Username == "" {
		return "", errors2.APIError{
			Code:    http.StatusUnauthorized,
			Message: "username is required",
		}
	}

	if reqBody.Token == "" {
		return reqBody.Username, errors2.APIError{
			Code:    http.StatusUnauthorized,
			Message: "token is required",
		}
	}

	return reqBody.Username, al.twoFASrv.ValidateToken(reqBody.Username, reqBody.Token)
}

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
	}

	response := api.NewSuccessPayload(map[string]interface{}{
		"version":                chshare.BuildVersion,
		"clients_connected":      countActive,
		"clients_disconnected":   countDisconnected,
		"fingerprint":            al.fingerprint,
		"connect_url":            al.config.Server.URL,
		"clients_auth_source":    al.clientAuthProvider.Source(),
		"clients_auth_mode":      al.getClientsAuthMode(),
		"users_auth_source":      al.usersService.ProviderType,
		"two_fa_enabled":         al.config.API.IsTwoFAOn(),
		"two_fa_delivery_method": twoFADelivery,
	})

	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleGetClients(w http.ResponseWriter, req *http.Request) {
	sortFunc, desc, err := getCorrespondingSortFunc(req.URL.Query().Get(queryParamSort))
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	clients, err := al.clientService.GetAll()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	sortFunc(clients, desc)

	clientsPayload := convertToClientsPayload(clients)
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(clientsPayload))
}

type UserPayload struct {
	Username    string   `json:"username"`
	Groups      []string `json:"groups"`
	TwoFASendTo string   `json:"two_fa_send_to"`
}

func (al *APIListener) handleGetUsers(w http.ResponseWriter, req *http.Request) {
	usrs, err := al.usersService.GetAll()
	if err != nil {
		al.jsonError(w, err)
		return
	}

	usersToSend := make([]UserPayload, 0, len(usrs))
	for i := range usrs {
		user := usrs[i]
		usersToSend = append(usersToSend, UserPayload{
			Username:    user.Username,
			Groups:      user.Groups,
			TwoFASendTo: user.TwoFASendTo,
		})
	}

	response := api.NewSuccessPayload(usersToSend)
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleChangeUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID, userIDExists := vars[routeParamUserID]
	if !userIDExists {
		userID = ""
	}

	var user users.User
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&user)
	if err == io.EOF { // is handled separately to return an informative error message
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Missing body with json data.")
		return
	} else if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid JSON data.", err)
		return
	}

	if err := al.usersService.Change(&user, userID); err != nil {
		al.jsonError(w, err)
		return
	}

	if userIDExists {
		al.Debugf("User [%s] updated.", userID)
		w.WriteHeader(http.StatusNoContent)
	} else {
		al.Debugf("User [%s] created.", user.Username)
		w.WriteHeader(http.StatusCreated)
	}
}

func (al *APIListener) handleDeleteUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID, userIDExists := vars[routeParamUserID]
	if !userIDExists {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Empty user id provided")
		return
	}

	if err := al.usersService.Delete(userID); err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("User [%s] deleted.", userID)
}

type ClientPayload struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	OS              string                  `json:"os"`
	OSArch          string                  `json:"os_arch"`
	OSFamily        string                  `json:"os_family"`
	OSKernel        string                  `json:"os_kernel"`
	Hostname        string                  `json:"hostname"`
	IPv4            []string                `json:"ipv4"`
	IPv6            []string                `json:"ipv6"`
	Tags            []string                `json:"tags"`
	Version         string                  `json:"version"`
	Address         string                  `json:"address"`
	Tunnels         []*clients.Tunnel       `json:"tunnels"`
	DisconnectedAt  *time.Time              `json:"disconnected_at"`
	ConnectionState clients.ConnectionState `json:"connection_state"`
	ClientAuthID    string                  `json:"client_auth_id"`
}

func convertToClientsPayload(clients []*clients.Client) []ClientPayload {
	r := make([]ClientPayload, 0, len(clients))
	for _, cur := range clients {
		r = append(r, ClientPayload{
			ID:              cur.ID,
			Name:            cur.Name,
			OS:              cur.OS,
			OSArch:          cur.OSArch,
			OSFamily:        cur.OSFamily,
			OSKernel:        cur.OSKernel,
			Hostname:        cur.Hostname,
			IPv4:            cur.IPv4,
			IPv6:            cur.IPv6,
			Tags:            cur.Tags,
			Version:         cur.Version,
			Address:         cur.Address,
			Tunnels:         cur.Tunnels,
			DisconnectedAt:  cur.DisconnectedAt,
			ConnectionState: cur.ConnectionState(),
			ClientAuthID:    cur.ClientAuthID,
		})
	}
	return r
}

func getCorrespondingSortFunc(sortStr string) (sortFunc func(a []*clients.Client, desc bool), desc bool, err error) {
	var sortField string
	if strings.HasPrefix(sortStr, "-") {
		desc = true
		sortField = sortStr[1:]
	} else {
		sortField = sortStr
	}

	switch sortField {
	case "":
		sortFunc = clients.SortByID
	case "id":
		sortFunc = clients.SortByID
	case "name":
		sortFunc = clients.SortByName
	case "os":
		sortFunc = clients.SortByOS
	case "hostname":
		sortFunc = clients.SortByHostname
	case "version":
		sortFunc = clients.SortByVersion
	default:
		err = fmt.Errorf("incorrect format of %q query param", queryParamSort)
	}
	return
}

func (al *APIListener) handleDeleteClient(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID := vars[routeParamClientID]
	err := al.clientService.DeleteOffline(clientID)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("Client %q deleted.", clientID)
}

const (
	URISchemeMaxLength = 15

	idleTimeoutMinutesQueryParam = "idle-timeout-minutes"
	idleTimeoutMin               = 0
	idleTimeoutMax               = 7 * 24 * 60 // week

	ErrCodeLocalPortInUse        = "ERR_CODE_LOCAL_PORT_IN_USE"
	ErrCodeRemotePortNotOpen     = "ERR_CODE_REMOTE_PORT_NOT_OPEN"
	ErrCodeTunnelExist           = "ERR_CODE_TUNNEL_EXIST"
	ErrCodeTunnelToPortExist     = "ERR_CODE_TUNNEL_TO_PORT_EXIST"
	ErrCodeURISchemeLengthExceed = "ERR_CODE_URI_SCHEME_LENGTH_EXCEED"
	ErrCodeInvalidACL            = "ERR_CODE_INVALID_ACL"
	ErrCodeInvalidIdleTimeout    = "ERR_CODE_INVALID_IDLE_TIMEOUT"
)

func (al *APIListener) handlePutClientTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID, exists := vars[routeParamClientID]
	if !exists || clientID == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "client id is missing")
		return
	}

	client, err := al.clientService.GetActiveByID(clientID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %s not found", clientID))
		return
	}

	localAddr := req.URL.Query().Get("local")
	remoteAddr := req.URL.Query().Get("remote")
	remoteStr := localAddr + ":" + remoteAddr
	if localAddr == "" {
		remoteStr = remoteAddr
	}
	remote, err := chshare.DecodeRemote(remoteStr)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("failed to decode %q: %v", remoteStr, err))
		return
	}

	idleTimeoutMinutesStr := req.URL.Query().Get(idleTimeoutMinutesQueryParam)
	var idleTimeoutMinutes int
	if idleTimeoutMinutesStr != "" {
		idleTimeoutMinutes, err = strconv.Atoi(idleTimeoutMinutesStr)
		if err != nil {
			al.jsonErrorResponseWithError(w, http.StatusBadRequest, ErrCodeInvalidIdleTimeout, fmt.Sprintf("invalid %q param", idleTimeoutMinutesQueryParam), err)
			return
		}

		if idleTimeoutMin > idleTimeoutMinutes || idleTimeoutMinutes > idleTimeoutMax {
			al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeInvalidIdleTimeout, fmt.Sprintf("%q param should be in range [%d,%d]", idleTimeoutMinutesQueryParam, idleTimeoutMin, idleTimeoutMax))
			return
		}
	}
	remote.IdleTimeoutMinutes = idleTimeoutMinutes

	aclStr := req.URL.Query().Get("acl")
	if _, err = clients.ParseTunnelACL(aclStr); err != nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeInvalidACL, fmt.Sprintf("Invalid ACL: %s", err))
		return
	}
	if aclStr != "" {
		remote.ACL = &aclStr
	}

	schemeStr := req.URL.Query().Get("scheme")
	if len(schemeStr) > URISchemeMaxLength {
		al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, ErrCodeURISchemeLengthExceed, "Invalid URI scheme.", "Exceeds the max length.")
		return
	}
	if schemeStr != "" {
		remote.Scheme = &schemeStr
	}

	if existing := client.FindTunnelByRemote(remote); existing != nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeTunnelExist, "Tunnel already exist.")
		return
	}

	for _, t := range client.Tunnels {
		if t.Remote.Remote() == remote.Remote() && t.EqualACL(remote.ACL) {
			al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeTunnelToPortExist, fmt.Sprintf("Tunnel to port %s already exist.", remote.RemotePort))
			return
		}
	}

	if checkPortStr := req.URL.Query().Get("check_port"); checkPortStr != "0" {
		if !al.checkRemotePort(w, *remote, client.Connection) {
			return
		}
	}

	// make next steps thread-safe
	client.Lock()
	defer client.Unlock()

	if remote.IsLocalSpecified() && !al.checkLocalPort(w, remote.LocalPort) {
		return
	}

	tunnels, err := al.clientService.StartClientTunnels(client, []*chshare.Remote{remote})
	if err != nil {
		al.jsonErrorResponse(w, http.StatusConflict, fmt.Errorf("can't create tunnel: %s", err))
		return
	}
	response := api.NewSuccessPayload(tunnels[0])
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) checkLocalPort(w http.ResponseWriter, localPort string) bool {
	lport, err := strconv.Atoi(localPort)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", fmt.Sprintf("Invalid port: %s.", localPort), err)
		return false
	}

	busyPorts, err := ports.ListBusyPorts()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return false
	}

	if busyPorts.Contains(lport) {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeLocalPortInUse, fmt.Sprintf("Port %d already in use.", lport))
		return false
	}

	return true
}

func (al *APIListener) checkRemotePort(w http.ResponseWriter, remote chshare.Remote, conn ssh.Conn) bool {
	req := &comm.CheckPortRequest{
		HostPort: remote.Remote(),
		Timeout:  al.config.Server.CheckPortTimeout,
	}
	resp := &comm.CheckPortResponse{}
	err := comm.SendRequestAndGetResponse(conn, comm.RequestTypeCheckPort, req, resp)
	if err != nil {
		if _, ok := err.(*comm.ClientError); ok {
			al.jsonErrorResponse(w, http.StatusConflict, err)
		} else {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		}
		return false
	}

	if !resp.Open {
		al.jsonErrorResponseWithDetail(
			w,
			http.StatusBadRequest,
			ErrCodeRemotePortNotOpen,
			fmt.Sprintf("Port %s is not in listening state.", remote.RemotePort),
			resp.ErrMsg,
		)
		return false
	}

	return true
}

func (al *APIListener) handleDeleteClientTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID, exists := vars[routeParamClientID]
	if !exists || clientID == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "client id is missing")
		return
	}

	force := false
	forceStr := req.URL.Query().Get("force")
	if forceStr != "" {
		var err error
		force, err = strconv.ParseBool(forceStr)
		if err != nil {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Invalid force param: %v.", forceStr))
			return
		}
	}

	client, err := al.clientService.GetActiveByID(clientID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %s not found", clientID))
		return
	}

	tunnelID, exists := vars["tunnel_id"]
	if !exists || tunnelID == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "tunnel id is missing")
		return
	}

	// make next steps thread-safe
	client.Lock()
	defer client.Unlock()

	tunnel := client.FindTunnel(tunnelID)
	if tunnel == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "tunnel not found")
		return
	}

	err = client.TerminateTunnel(tunnel, force)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusConflict, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetMe returns the currently logged in user and the groups the user belongs to.
func (al *APIListener) handleGetMe(w http.ResponseWriter, req *http.Request) {
	user, err := al.getUserModel(req)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}

	me := UserPayload{
		Username:    user.Username,
		Groups:      user.Groups,
		TwoFASendTo: user.TwoFASendTo,
	}
	response := api.NewSuccessPayload(me)
	al.writeJSONResponse(w, http.StatusOK, response)
}

type changeMeRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	OldPassword string `json:"old_password"`
	TwoFASendTo string `json:"two_fa_send_to"`
}

func (al *APIListener) handleChangeMe(w http.ResponseWriter, req *http.Request) {
	var r changeMeRequest
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&r)
	if err == io.EOF { // is handled separately to return an informative error message
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Missing body with json data.")
		return
	} else if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid JSON data.", err)
		return
	}

	curUser, err := al.getUserModelForAuth(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	if r.Password != "" {
		if r.OldPassword == "" {
			al.jsonErrorResponseWithTitle(w, http.StatusForbidden, "Missing old password.")
			return
		}

		if !verifyPassword(*curUser, r.OldPassword) {
			al.jsonErrorResponseWithTitle(w, http.StatusForbidden, "Incorrect old password.")
			return
		}
	}

	if err := al.usersService.Change(&users.User{
		Username:    r.Username,
		Password:    r.Password,
		TwoFASendTo: r.TwoFASendTo,
	}, curUser.Username); err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) getUserModel(req *http.Request) (*users.User, error) {
	curUsername := api.GetUser(req.Context(), al.Logger)
	if curUsername == "" {
		return nil, nil
	}

	user, err := al.userSrv.GetByUsername(curUsername)
	if err != nil {
		return nil, err
	}

	return user, err
}

func (al *APIListener) getUserModelForAuth(req *http.Request) (*users.User, error) {
	usr, err := al.getUserModel(req)
	if err != nil {
		return nil, errors2.APIError{
			Err:  err,
			Code: http.StatusInternalServerError,
		}
	}

	if usr == nil {
		return nil, errors2.APIError{
			Message: "unauthorized access",
			Code:    http.StatusUnauthorized,
		}
	}

	return usr, nil
}

func (al *APIListener) handleGetIP(w http.ResponseWriter, req *http.Request) {
	ipResp := struct {
		IP string `json:"ip"`
	}{
		IP: realip.FromRequest(req),
	}
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(ipResp))
}

const (
	MinCredentialsLength = 3

	ErrCodeClientAuthSingleClient = "ERR_CODE_CLIENT_AUTH_SINGLE"
	ErrCodeClientAuthRO           = "ERR_CODE_CLIENT_AUTH_RO"

	ErrCodeClientAuthHasClient = "ERR_CODE_CLIENT_AUTH_HAS_CLIENT"
	ErrCodeClientAuthNotFound  = "ERR_CODE_CLIENT_AUTH_NOT_FOUND"
)

func (al *APIListener) handleGetClientsAuth(w http.ResponseWriter, req *http.Request) {
	rClients, err := al.clientAuthProvider.GetAll()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	clientsauth.SortByID(rClients, false)

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(rClients))
}

func (al *APIListener) handlePostClientsAuth(w http.ResponseWriter, req *http.Request) {
	if !al.allowClientAuthWrite(w) {
		return
	}

	var newClient clientsauth.ClientAuth
	err := json.NewDecoder(req.Body).Decode(&newClient)
	if err == io.EOF {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Missing data.")
		return
	} else if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON data.", err)
		return
	}

	if len(newClient.ID) < MinCredentialsLength {
		al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid or missing ID.", fmt.Sprintf("Min size is %d.", MinCredentialsLength))
		return
	}

	if len(newClient.Password) < MinCredentialsLength {
		al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid or missing password.", fmt.Sprintf("Min size is %d.", MinCredentialsLength))
		return
	}

	added, err := al.clientAuthProvider.Add(&newClient)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if !added {
		al.jsonErrorResponseWithDetail(w, http.StatusConflict, ErrCodeAlreadyExist, fmt.Sprintf("Client Auth with ID %q already exist.", newClient.ID), "")
		return
	}

	al.Infof("ClientAuth %q created.", newClient.ID)

	w.WriteHeader(http.StatusCreated)
}

func (al *APIListener) handleDeleteClientAuth(w http.ResponseWriter, req *http.Request) {
	if !al.allowClientAuthWrite(w) {
		return
	}

	vars := mux.Vars(req)
	clientAuthID := vars["client_auth_id"]
	if clientAuthID == "" {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeMissingRouteVar, "Missing 'client_auth_id' route param.")
		return
	}

	force := false
	forceStr := req.URL.Query().Get("force")
	if forceStr != "" {
		var err error
		force, err = strconv.ParseBool(forceStr)
		if err != nil {
			al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeInvalidRequest, fmt.Sprintf("Invalid force param %v.", forceStr))
			return
		}
	}

	existing, err := al.clientAuthProvider.Get(clientAuthID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if existing == nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusNotFound, ErrCodeClientAuthNotFound, fmt.Sprintf("Client Auth with ID=%q not found.", clientAuthID))
		return
	}

	allClients := al.clientService.GetAllByClientID(clientAuthID)
	if !force && len(allClients) > 0 {
		al.jsonErrorResponseWithErrCode(w, http.StatusConflict, ErrCodeClientAuthHasClient, fmt.Sprintf("Client Auth expected to have no active or disconnected bound client(s), got %d.", len(allClients)))
		return
	}

	for _, s := range allClients {
		if err := al.clientService.ForceDelete(s); err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
	}

	err = al.clientAuthProvider.Delete(clientAuthID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	al.Infof("ClientAuth %q deleted.", clientAuthID)

	w.WriteHeader(http.StatusNoContent)
}

type clientsAuthMode string

const (
	clientsAuthModeRO = "Read Only"
	clientsAuthModeRW = "Read Write"
)

func (al *APIListener) getClientsAuthMode() clientsAuthMode {
	if al.isClientsAuthWriteable() {
		return clientsAuthModeRW
	}
	return clientsAuthModeRO
}

func (al *APIListener) isClientsAuthWriteable() bool {
	return al.clientAuthProvider.IsWriteable() && al.config.Server.AuthWrite
}

func (al *APIListener) allowClientAuthWrite(w http.ResponseWriter) bool {
	if !al.clientAuthProvider.IsWriteable() {
		al.jsonErrorResponseWithErrCode(w, http.StatusMethodNotAllowed, ErrCodeClientAuthSingleClient, "Client authentication is enabled only for a single user.")
		return false
	}

	if !al.config.Server.AuthWrite {
		al.jsonErrorResponseWithErrCode(w, http.StatusMethodNotAllowed, ErrCodeClientAuthRO, "Client authentication has been attached in read-only mode.")
		return false
	}

	return true
}

func (al *APIListener) handlePostCommand(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cid := vars[routeParamClientID]
	if cid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamClientID))
		return
	}

	reqBody := &api.ExecuteCommandInput{}
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&reqBody)
	if err == io.EOF { // is handled separately to return an informative error message
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Missing body with json data.")
		return
	} else if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid JSON data.", err)
		return
	}
	reqBody.ClientID = cid

	al.handleExecuteCommand(req.Context(), w, reqBody)
}

func (al *APIListener) handleExecuteCommand(ctx context.Context, w http.ResponseWriter, reqBody *api.ExecuteCommandInput) {
	if reqBody.Command == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Command cannot be empty.")
		return
	}
	if err := validateShell(reqBody.Shell); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid shell.", err)
		return
	}

	if reqBody.TimeoutSec <= 0 {
		reqBody.TimeoutSec = al.config.Server.RunRemoteCmdTimeoutSec
	}

	client, err := al.clientService.GetActiveByID(reqBody.ClientID)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to find an active client with id=%q.", reqBody.ClientID), err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Active client with id=%q not found.", reqBody.ClientID))
		return
	}

	// send the command to the client
	// Send a job with all possible info in order to get the full-populated job back (in client-listener) when it's done.
	// Needed when server restarts to get all job data from client. Because on server restart job running info is lost.
	curJob := models.Job{
		JobSummary: models.JobSummary{
			JID:        generateNewJobID(),
			FinishedAt: nil,
		},
		ClientID:   reqBody.ClientID,
		ClientName: client.Name,
		Command:    reqBody.Command,
		Shell:      reqBody.Shell,
		CreatedBy:  api.GetUser(ctx, al.Logger),
		TimeoutSec: reqBody.TimeoutSec,
		Result:     nil,
		Cwd:        reqBody.Cwd,
		IsSudo:     reqBody.IsSudo,
		IsScript:   reqBody.IsScript,
	}
	sshResp := &comm.RunCmdResponse{}
	err = comm.SendRequestAndGetResponse(client.Connection, comm.RequestTypeRunCmd, curJob, sshResp)
	if err != nil {
		if _, ok := err.(*comm.ClientError); ok {
			al.jsonErrorResponseWithTitle(w, http.StatusConflict, err.Error())
		} else {
			al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to execute remote command.", err)
		}
		return
	}

	// set fields received in response
	curJob.PID = &sshResp.Pid
	curJob.StartedAt = sshResp.StartedAt
	curJob.Status = models.JobStatusRunning

	if err := al.jobProvider.CreateJob(&curJob); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to persist a new job.", err)
		return
	}

	resp := struct {
		JID string `json:"jid"`
	}{
		JID: curJob.JID,
	}
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(resp))

	al.Debugf("Job[id=%q] created to execute remote command on client with id=%q: %q.", curJob.JID, reqBody.ClientID, reqBody.Command)
}

func (al *APIListener) createScriptExecutionInputFromRequest(req *http.Request) (*script.ExecutionInput, error) {
	var err error
	vars := mux.Vars(req)
	query := req.URL.Query()
	clientID := vars[routeParamClientID]
	clientIDFromQuery, clientIDFound := query[routeParamClientID]
	if clientID == "" && !clientIDFound && len(clientIDFromQuery) == 0 {
		return nil, errors2.APIError{
			Message: "Missing client id",
			Code:    http.StatusBadRequest,
		}
	}

	if clientID == "" {
		clientID = clientIDFromQuery[0]
	}

	isPowershell := false
	isPowershellParam, ok := query[isPowershellScriptParam]
	if ok && len(isPowershellParam) > 0 && isPowershellParam[0] != "" && isPowershellParam[0] != "false" {
		isPowershell = true
	}

	isSudo := false
	isSudoParam, ok := query[isSudoScriptParam]
	if ok && len(isSudoParam) > 0 && isSudoParam[0] != "" && isSudoParam[0] != "false" {
		isSudo = true
	}

	cwd := ""
	cwdParam, ok := query[cwdScriptParam]
	if ok && len(cwdParam) > 0 {
		cwd = cwdParam[0]
	}

	timeout := time.Duration(al.config.Server.RunRemoteCmdTimeoutSec) * time.Second
	timeoutParam, ok := query[timeoutScriptParam]
	if ok && len(timeoutParam) > 0 {
		timeoutStr := timeoutParam[0]
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, errors2.APIError{
				Message: fmt.Sprintf("Failed to parse timeout value %s", timeoutScriptParam),
				Err:     err,
				Code:    http.StatusBadRequest,
			}
		}
	}

	client, err := al.clientService.GetActiveByID(clientID)
	if err != nil {
		return nil, errors2.APIError{
			Message: fmt.Sprintf("Failed to find an active client with id=%q.", clientID),
			Err:     err,
			Code:    http.StatusInternalServerError,
		}
	}
	if client == nil {
		return nil, errors2.APIError{
			Message: fmt.Sprintf("Active client with id=%q not found.", clientID),
			Code:    http.StatusNotFound,
		}
	}

	if client.Version != chshare.SourceVersion && client.Version < minVersionScriptExecSupport {
		return nil, errors2.APIError{
			Message: fmt.Sprintf("Script Execution is supported starting from %s version, current client version is %s.", minVersionScriptExecSupport, client.Version),
			Code:    http.StatusBadRequest,
		}
	}

	return &script.ExecutionInput{
		Client:       client,
		IsSudo:       isSudo,
		IsPowershell: isPowershell,
		Cwd:          cwd,
		Timeout:      timeout,
	}, nil
}

func (al *APIListener) handleExecuteScript(w http.ResponseWriter, req *http.Request) {
	scriptInput, err := al.createScriptExecutionInputFromRequest(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	scriptBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to get script body", err)
		return
	}
	scriptInput.ScriptBody = scriptBody

	scriptPath, err := al.scriptManager.CreateScriptOnClient(scriptInput)
	if err != nil {
		if _, ok := err.(*comm.ClientError); ok {
			al.jsonErrorResponseWithTitle(w, http.StatusConflict, err.Error())
		} else {
			al.jsonError(w, err)
		}
		return
	}

	cmdInput := al.scriptManager.ConvertScriptInputToCmdInput(scriptInput, scriptPath)

	al.handleExecuteCommand(req.Context(), w, cmdInput)
}

func validateShell(shell string) error {
	if shell == "" {
		return nil
	}
	for _, v := range validInputShell {
		if shell == v {
			return nil
		}
	}
	return fmt.Errorf("expected shell to be one of: %s, actual: %s", validInputShell, shell)
}

func (al *APIListener) handleGetCommands(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cid := vars[routeParamClientID]
	if cid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamClientID))
		return
	}

	res, err := al.jobProvider.GetSummariesByClientID(cid)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to get client jobs: client_id=%q.", cid), err)
		return
	}

	jobs.SortByFinishedAt(res, true)
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(res))
}

func (al *APIListener) handleGetCommand(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cid := vars[routeParamClientID]
	if cid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamClientID))
		return
	}
	jid := vars[routeParamJobID]
	if jid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamJobID))
		return
	}

	job, err := al.jobProvider.GetByJID(cid, jid)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to find a job[id=%q].", jid), err)
		return
	}
	if job == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Job[id=%q] not found.", jid))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(job))
}

type newJobResponse struct {
	JID string `json:"jid"`
}

type multiClientCmdRequest struct {
	ClientIDs           []string `json:"client_ids"`
	GroupIDs            []string `json:"group_ids"`
	Command             string   `json:"command"`
	Cwd                 string   `json:"cwd"`
	IsSudo              bool     `json:"sudo"`
	Shell               string   `json:"shell"`
	TimeoutSec          int      `json:"timeout_sec"`
	ExecuteConcurrently bool     `json:"execute_concurrently"`
	AbortOnError        *bool    `json:"abort_on_error"` // pointer is used because it's default value is true. Otherwise it would be more difficult to check whether this field is missing or not
	IsScript            bool
}

// TODO: refactor to reuse similar code for REST API and WebSocket to execute cmds if both will be supported
func (al *APIListener) handlePostMultiClientCommand(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqBody := multiClientCmdRequest{}
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&reqBody)
	if err == io.EOF { // is handled separately to return an informative error message
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Missing body with json data.")
		return
	} else if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid JSON data.", err)
		return
	}
	if reqBody.Command == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Command cannot be empty.")
		return
	}
	if err := validateShell(reqBody.Shell); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid shell.", err)
		return
	}

	if reqBody.TimeoutSec <= 0 {
		reqBody.TimeoutSec = al.config.Server.RunRemoteCmdTimeoutSec
	}

	var groups []*cgroups.ClientGroup
	for _, groupID := range reqBody.GroupIDs {
		group, err := al.clientGroupProvider.Get(ctx, groupID)
		if err != nil {
			al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to get a client group with id=%q.", groupID), err)
			return
		}
		if group == nil {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Unknown group with id=%q.", groupID))
			return
		}
		groups = append(groups, group)
	}
	groupClients := al.clientService.GetActiveByGroups(groups)

	if len(reqBody.GroupIDs) > 0 && len(groupClients) == 0 && len(reqBody.ClientIDs) == 0 {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "No active clients belong to the selected group(s).")
		return
	}

	minClients := 2
	if len(reqBody.ClientIDs) < minClients && len(groupClients) == 0 {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("At least %d clients should be specified.", minClients))
		return
	}

	var orderedClients []*clients.Client
	usedClientIDs := make(map[string]bool)
	for _, cid := range reqBody.ClientIDs {
		client, err := al.clientService.GetByID(cid)
		if err != nil {
			al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to find a client with id=%q.", cid), err)
			return
		}
		if client == nil {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Client with id=%q not found.", cid))
			return
		}
		if client.DisconnectedAt != nil {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Client with id=%q is not active.", cid))
			return
		}
		usedClientIDs[cid] = true
		orderedClients = append(orderedClients, client)
	}

	// append group clients
	for _, groupClient := range groupClients {
		if !usedClientIDs[groupClient.ID] {
			usedClientIDs[groupClient.ID] = true
			orderedClients = append(orderedClients, groupClient)
		}
	}

	// by default abortOnErr is true
	abortOnErr := true
	if reqBody.AbortOnError != nil {
		abortOnErr = *reqBody.AbortOnError
	}

	multiJob := &models.MultiJob{
		MultiJobSummary: models.MultiJobSummary{
			JID:       generateNewJobID(),
			StartedAt: time.Now(),
			CreatedBy: api.GetUser(req.Context(), al.Logger),
		},
		ClientIDs:  reqBody.ClientIDs,
		GroupIDs:   reqBody.GroupIDs,
		Command:    reqBody.Command,
		Shell:      reqBody.Shell,
		Cwd:        reqBody.Cwd,
		IsSudo:     reqBody.IsSudo,
		TimeoutSec: reqBody.TimeoutSec,
		Concurrent: reqBody.ExecuteConcurrently,
		AbortOnErr: abortOnErr,
	}
	if err := al.jobProvider.SaveMultiJob(multiJob); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to persist a new multi-client job.", err)
		return
	}

	resp := newJobResponse{
		JID: multiJob.JID,
	}
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(resp))

	al.Debugf("Multi-client Job[id=%q] created to execute remote command on clients %s, groups %s: %q.", multiJob.JID, reqBody.ClientIDs, reqBody.GroupIDs, reqBody.Command)

	go al.executeMultiClientJob(multiJob, orderedClients)
}

func (al *APIListener) executeMultiClientJob(job *models.MultiJob, orderedClients []*clients.Client) {
	// for sequential execution - create a channel to get the job result
	var curJobDoneChannel chan *models.Job
	if !job.Concurrent {
		curJobDoneChannel = make(chan *models.Job)
		al.jobsDoneChannel.Set(job.JID, curJobDoneChannel)
		defer func() {
			close(curJobDoneChannel)
			al.jobsDoneChannel.Del(job.JID)
		}()
	}
	for _, client := range orderedClients {
		if job.Concurrent {
			go al.createAndRunJob(
				job.JID,
				job.Command,
				job.Shell,
				job.CreatedBy,
				job.Cwd,
				job.TimeoutSec,
				job.IsSudo,
				job.IsScript,
				client,
			)
		} else {
			success := al.createAndRunJob(
				job.JID,
				job.Command,
				job.Shell,
				job.CreatedBy,
				job.Cwd,
				job.TimeoutSec,
				job.IsSudo,
				job.IsScript,
				client,
			)
			if !success {
				if job.AbortOnErr {
					break
				}
				continue
			}

			// in tests skip next part to avoid waiting
			if al.insecureForTests {
				continue
			}

			// wait until command is finished
			jobResult := <-curJobDoneChannel
			if job.AbortOnErr && jobResult.Status == models.JobStatusFailed {
				break
			}
		}
	}
	if al.testDone != nil {
		al.testDone <- true
	}
}

func (al *APIListener) createAndRunJob(
	jid, cmd, shell, createdBy, cwd string,
	timeoutSec int,
	isSudo, isScript bool,
	client *clients.Client,
) bool {
	// send the command to the client
	curJob := models.Job{
		JobSummary: models.JobSummary{
			JID: generateNewJobID(),
		},
		StartedAt:  time.Now(),
		ClientID:   client.ID,
		ClientName: client.Name,
		Command:    cmd,
		Cwd:        cwd,
		IsSudo:     isSudo,
		IsScript:   isScript,
		Shell:      shell,
		CreatedBy:  createdBy,
		TimeoutSec: timeoutSec,
		MultiJobID: &jid,
	}
	sshResp := &comm.RunCmdResponse{}
	err := comm.SendRequestAndGetResponse(client.Connection, comm.RequestTypeRunCmd, curJob, sshResp)
	// return an error after saving the job
	if err != nil {
		// failure, set fields to mark it as failed
		al.Errorf("multi_client_id=%q, client_id=%q, Error on execute remote command: %v", *curJob.MultiJobID, curJob.ClientID, err)
		curJob.Status = models.JobStatusFailed
		now := time.Now()
		curJob.FinishedAt = &now
		curJob.Error = err.Error()
	} else {
		// success, set fields received in response
		curJob.PID = &sshResp.Pid
		curJob.StartedAt = sshResp.StartedAt // override with the start time of the command
		curJob.Status = models.JobStatusRunning
	}

	if dbErr := al.jobProvider.CreateJob(&curJob); dbErr != nil {
		// just log it, cmd is running, when it's finished it can be saved on result return
		al.Errorf("multi_client_id=%q, client_id=%q, Failed to persist a child job: %v", *curJob.MultiJobID, curJob.ClientID, dbErr)
	}

	return err == nil
}

func (al *APIListener) handleCommandsWS(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	uiConn, err := apiUpgrader.Upgrade(w, req, nil)
	if err != nil {
		al.Errorf("Failed to establish WS connection: %v", err)
		return
	}
	uiConnTS := ws.NewConcurrentWebSocket(uiConn, al.Logger)
	inboundMsg := &multiClientCmdRequest{}
	err = uiConnTS.ReadJSON(inboundMsg)
	if err == io.EOF { // is handled separately to return an informative error message
		uiConnTS.WriteError("Inbound message should contain non empty json object with command data.", nil)
		return
	} else if err != nil {
		uiConnTS.WriteError("Invalid JSON data.", err)
		return
	}

	al.handleCommandsExecutionWS(ctx, uiConnTS, inboundMsg)
}

func (al *APIListener) handleScriptsWS(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	uiConn, err := apiUpgrader.Upgrade(w, req, nil)
	if err != nil {
		al.Errorf("Failed to establish WS connection: %v", err)
		return
	}

	uiConnTS := ws.NewConcurrentWebSocket(uiConn, al.Logger)
	_, scriptBody, err := uiConnTS.ReadMessage()
	if err != nil {
		uiConnTS.WriteError("Script message should contain non empty data.", nil)
		return
	}

	scriptInput, err := al.createScriptExecutionInputFromRequest(req)
	if err != nil {
		uiConnTS.WriteError("Request failure", err)
		return
	}
	scriptInput.ScriptBody = scriptBody

	scriptPath, err := al.scriptManager.CreateScriptOnClient(scriptInput)
	if err != nil {
		uiConnTS.WriteError("Script creation failure", err)
		return
	}

	cmdInput := al.scriptManager.ConvertScriptInputToCmdInput(scriptInput, scriptPath)

	abortOnErr := true
	wsCmdRequest := &multiClientCmdRequest{
		ClientIDs:           []string{scriptInput.Client.ID},
		Command:             cmdInput.Command,
		Cwd:                 cmdInput.Cwd,
		IsSudo:              cmdInput.IsSudo,
		Shell:               cmdInput.Shell,
		TimeoutSec:          cmdInput.TimeoutSec,
		ExecuteConcurrently: false,
		AbortOnError:        &abortOnErr,
		IsScript:            cmdInput.IsScript,
	}

	al.handleCommandsExecutionWS(ctx, uiConnTS, wsCmdRequest)
}

func (al *APIListener) handleCommandsExecutionWS(ctx context.Context, uiConnTS *ws.ConcurrentWebSocket, inboundMsg *multiClientCmdRequest) {
	if inboundMsg.Command == "" {
		uiConnTS.WriteError("Command cannot be empty.", nil)
		return
	}
	if err := validateShell(inboundMsg.Shell); err != nil {
		uiConnTS.WriteError("Invalid shell.", err)
		return
	}

	if inboundMsg.TimeoutSec <= 0 {
		inboundMsg.TimeoutSec = al.config.Server.RunRemoteCmdTimeoutSec
	}

	var groups []*cgroups.ClientGroup
	for _, groupID := range inboundMsg.GroupIDs {
		group, err := al.clientGroupProvider.Get(ctx, groupID)
		if err != nil {
			uiConnTS.WriteError(fmt.Sprintf("Failed to get a client group with id=%q.", groupID), err)
			return
		}
		if group == nil {
			uiConnTS.WriteError(fmt.Sprintf("Unknown group with id=%q.", groupID), nil)
			return
		}
		groups = append(groups, group)
	}
	groupClients := al.clientService.GetActiveByGroups(groups)

	if len(inboundMsg.GroupIDs) > 0 && len(groupClients) == 0 && len(inboundMsg.ClientIDs) == 0 {
		uiConnTS.WriteError("No active clients belong to the selected group(s).", nil)
		return
	}

	if len(inboundMsg.ClientIDs) < 1 && len(groupClients) == 0 {
		uiConnTS.WriteError("'client_ids' field should contain at least one client ID", nil)
		return
	}

	var orderedClients []*clients.Client
	usedClientIDs := make(map[string]bool)
	for _, cid := range inboundMsg.ClientIDs {
		client, err := al.clientService.GetByID(cid)
		if err != nil {
			uiConnTS.WriteError(fmt.Sprintf("Failed to find a client with id=%q.", cid), err)
			return
		}
		if client == nil {
			uiConnTS.WriteError(fmt.Sprintf("Client with id=%q not found.", cid), nil)
			return
		}
		if client.DisconnectedAt != nil {
			uiConnTS.WriteError(fmt.Sprintf("Client with id=%q is not active.", cid), nil)
			return
		}
		usedClientIDs[cid] = true
		orderedClients = append(orderedClients, client)
	}

	// append group clients
	for _, groupClient := range groupClients {
		if !usedClientIDs[groupClient.ID] {
			usedClientIDs[groupClient.ID] = true
			orderedClients = append(orderedClients, groupClient)
		}
	}

	jid := generateNewJobID()
	al.Server.uiJobWebSockets.Set(jid, uiConnTS)
	defer al.Server.uiJobWebSockets.Delete(jid)

	createdBy := api.GetUser(ctx, al.Logger)
	if len(inboundMsg.ClientIDs) > 1 || len(groupClients) > 0 {
		// by default abortOnErr is true
		abortOnErr := true
		if inboundMsg.AbortOnError != nil {
			abortOnErr = *inboundMsg.AbortOnError
		}

		multiJob := &models.MultiJob{
			MultiJobSummary: models.MultiJobSummary{
				JID:       jid,
				StartedAt: time.Now(),
				CreatedBy: createdBy,
			},
			ClientIDs:  inboundMsg.ClientIDs,
			GroupIDs:   inboundMsg.GroupIDs,
			Command:    inboundMsg.Command,
			Cwd:        inboundMsg.Cwd,
			Shell:      inboundMsg.Shell,
			TimeoutSec: inboundMsg.TimeoutSec,
			Concurrent: inboundMsg.ExecuteConcurrently,
			AbortOnErr: abortOnErr,
			IsSudo:     inboundMsg.IsSudo,
			IsScript:   inboundMsg.IsScript,
		}
		if err := al.jobProvider.SaveMultiJob(multiJob); err != nil {
			uiConnTS.WriteError("Failed to persist a new multi-client job.", err)
			return
		}

		al.Debugf("Multi-client Job[id=%q] created to execute remote command on clients %s, groups %s: %q.", multiJob.JID, inboundMsg.ClientIDs, inboundMsg.GroupIDs, inboundMsg.Command)
		uiConnTS.SetWritesBeforeClose(len(orderedClients))

		// for sequential execution - create a channel to get the job result
		var curJobDoneChannel chan *models.Job
		if !multiJob.Concurrent {
			curJobDoneChannel = make(chan *models.Job)
			al.jobsDoneChannel.Set(multiJob.JID, curJobDoneChannel)
			defer func() {
				close(curJobDoneChannel)
				al.jobsDoneChannel.Del(multiJob.JID)
			}()
		}

		for _, client := range orderedClients {
			curJID := generateNewJobID()
			if multiJob.Concurrent {
				go al.createAndRunJobWS(
					uiConnTS,
					&jid,
					curJID,
					multiJob.Command,
					multiJob.Shell,
					createdBy,
					multiJob.Cwd,
					multiJob.TimeoutSec,
					multiJob.IsSudo,
					multiJob.IsScript,
					client,
				)
			} else {
				success := al.createAndRunJobWS(
					uiConnTS,
					&jid,
					curJID,
					multiJob.Command,
					multiJob.Shell,
					createdBy,
					multiJob.Cwd,
					multiJob.TimeoutSec,
					multiJob.IsSudo,
					multiJob.IsScript,
					client,
				)
				if !success {
					if multiJob.AbortOnErr {
						uiConnTS.Close()
						return
					}
					continue
				}
				// wait until command is finished
				jobResult := <-curJobDoneChannel
				if multiJob.AbortOnErr && jobResult.Status == models.JobStatusFailed {
					uiConnTS.Close()
					return
				}
			}
		}
	} else {
		al.createAndRunJobWS(
			uiConnTS,
			nil,
			jid,
			inboundMsg.Command,
			inboundMsg.Shell,
			createdBy,
			inboundMsg.Cwd,
			inboundMsg.TimeoutSec,
			inboundMsg.IsSudo,
			inboundMsg.IsScript,
			orderedClients[0],
		)
	}

	// check for Close message from client to close the connection
	mt, message, err := uiConnTS.ReadMessage()
	if err != nil {
		if closeErr, ok := err.(*websocket.CloseError); ok {
			al.Debugf("Received a closed err on WS read: %v", closeErr)
			return
		}
		al.Debugf("Error read from websocket: %v", err)
		return
	}

	al.Debugf("Message received: type %v, msg %s", mt, message)
	uiConnTS.Close()
}

func (al *APIListener) createAndRunJobWS(
	uiConnTS *ws.ConcurrentWebSocket,
	multiJobID *string,
	jid, cmd, shell, createdBy, cwd string,
	timeoutSec int,
	isSudo, isScript bool,
	client *clients.Client,
) bool {
	curJob := models.Job{
		JobSummary: models.JobSummary{
			JID: jid,
		},
		StartedAt:  time.Now(),
		ClientID:   client.ID,
		ClientName: client.Name,
		Command:    cmd,
		Shell:      shell,
		CreatedBy:  createdBy,
		TimeoutSec: timeoutSec,
		MultiJobID: multiJobID,
		Cwd:        cwd,
		IsSudo:     isSudo,
		IsScript:   isScript,
	}
	logPrefix := curJob.LogPrefix()

	// send the command to the client
	sshResp := &comm.RunCmdResponse{}
	err := comm.SendRequestAndGetResponse(client.Connection, comm.RequestTypeRunCmd, curJob, sshResp)
	if err != nil {
		al.Errorf("%s, Error on execute remote command: %v", logPrefix, err)

		curJob.Status = models.JobStatusFailed
		now := time.Now()
		curJob.FinishedAt = &now
		curJob.Error = err.Error()

		// send the failed job to UI
		_ = uiConnTS.WriteJSON(curJob)
	} else {
		al.Debugf("%s, Job was sent to execute remote command: %q.", logPrefix, curJob.Command)

		// success, set fields received in response
		curJob.PID = &sshResp.Pid
		curJob.StartedAt = sshResp.StartedAt // override with the start time of the command
		curJob.Status = models.JobStatusRunning
	}

	// do not save the failed job if it's a single-client job
	if err != nil && multiJobID == nil {
		return false
	}

	if dbErr := al.jobProvider.CreateJob(&curJob); dbErr != nil {
		// just log it, cmd is running, when it's finished it can be saved on result return
		al.Errorf("%s, Failed to persist job: %v", logPrefix, dbErr)
	}

	return err == nil
}

func (al *APIListener) handleGetMultiClientCommand(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	jid := vars[routeParamJobID]
	if jid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamJobID))
		return
	}

	job, err := al.jobProvider.GetMultiJob(jid)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to find a multi-client job[id=%q].", jid), err)
		return
	}
	if job == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Multi-client Job[id=%q] not found.", jid))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(job))
}

func (al *APIListener) handleGetMultiClientCommands(w http.ResponseWriter, req *http.Request) {
	res, err := al.jobProvider.GetAllMultiJobSummaries()
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to get multi-client jobs.", err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(res))
}

func (al *APIListener) handlePostClientGroups(w http.ResponseWriter, req *http.Request) {
	var group cgroups.ClientGroup
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&group)
	if err == io.EOF { // is handled separately to return an informative error message
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Missing body with json data.")
		return
	} else if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid JSON data.", err)
		return
	}

	if err := validateInputClientGroup(group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid client group.", err)
		return
	}

	if err := al.clientGroupProvider.Create(req.Context(), &group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to persist a new client group.", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	al.Debugf("Client Group [id=%q] created.", group.ID)
}

func (al *APIListener) handlePutClientGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars[routeParamGroupID]
	if id == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamGroupID))
		return
	}

	var group cgroups.ClientGroup
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&group)
	if err == io.EOF { // is handled separately to return an informative error message
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Missing body with json data.")
		return
	} else if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid JSON data.", err)
		return
	}

	if id != group.ID {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("%q route param doesn't not match group ID from request body.", routeParamGroupID))
		return
	}

	if err := validateInputClientGroup(group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid client group.", err)
		return
	}

	if err := al.clientGroupProvider.Update(req.Context(), &group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to persist client group.", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("Client Group [id=%q] updated.", group.ID)
}

const groupIDMaxLength = 30
const validGroupIDChars = "A-Za-z0-9_-*"

var invalidGroupIDRegexp = regexp.MustCompile(`[^\*A-Za-z0-9_-]`)

func validateInputClientGroup(group cgroups.ClientGroup) error {
	if strings.TrimSpace(group.ID) == "" {
		return errors.New("group ID cannot be empty")
	}
	if len(group.ID) > groupIDMaxLength {
		return fmt.Errorf("invalid group ID: max length %d, got %d", groupIDMaxLength, len(group.ID))
	}
	if invalidGroupIDRegexp.MatchString(group.ID) {
		return fmt.Errorf("invalid group ID %q: can contain only %q", group.ID, validGroupIDChars)
	}
	return nil
}

func (al *APIListener) handleGetClientGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars[routeParamGroupID]
	if id == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamGroupID))
		return
	}

	group, err := al.clientGroupProvider.Get(req.Context(), id)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to find client group[id=%q].", id), err)
		return
	}
	if group == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Client Group[id=%q] not found.", id))
		return
	}

	al.clientService.PopulateGroupsWithClients([]*cgroups.ClientGroup{group})
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(group))
}

func (al *APIListener) handleGetClientGroups(w http.ResponseWriter, req *http.Request) {
	res, err := al.clientGroupProvider.GetAll(req.Context())
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to get client groups.", err)
		return
	}

	al.clientService.PopulateGroupsWithClients(res)
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(res))
}

func (al *APIListener) handleDeleteClientGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars[routeParamGroupID]
	if id == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamGroupID))
		return
	}

	err := al.clientGroupProvider.Delete(req.Context(), id)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to delete client group[id=%q].", id), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("Client Group [id=%q] deleted.", id)
}

func (al *APIListener) wrapStaticPassModeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if al.usersService.ProviderType == enums.ProviderSourceStatic {
			al.jsonError(w, errors2.APIError{
				Code:    http.StatusBadRequest,
				Message: "server runs on a static user-password pair, please use JSON file or database for user data",
			})
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (al *APIListener) wrapAdminAccessMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := al.getUserModelForAuth(r)
		if err != nil {
			al.jsonError(w, err)
			return
		}

		for i := range user.Groups {
			if user.Groups[i] == users.Administrators {
				next.ServeHTTP(w, r)
				return
			}
		}

		al.jsonError(w, errors2.APIError{
			Message: fmt.Sprintf(
				"current user should belong to %s group to access this resource",
				users.Administrators,
			),
			Code: http.StatusForbidden,
		})
	}
}

func (al *APIListener) handleGetVaultStatus(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	st, err := al.vaultManager.Status(ctx)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(st))
}

func (al *APIListener) handleVaultUnlock(w http.ResponseWriter, req *http.Request) {
	passReq, err := al.extractPassRequest(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.vaultManager.UnLock(req.Context(), passReq.Password)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (al *APIListener) handleVaultLock(w http.ResponseWriter, req *http.Request) {
	err := al.vaultManager.Lock(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) handleVaultInit(w http.ResponseWriter, req *http.Request) {
	passReq, err := al.extractPassRequest(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.vaultManager.Init(req.Context(), passReq.Password)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (al *APIListener) extractPassRequest(req *http.Request) (vault.PassRequest, error) {
	var passReq vault.PassRequest

	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&passReq)
	if err == io.EOF {
		return passReq, errors2.APIError{
			Message: "Missing password.",
			Code:    http.StatusBadRequest,
		}
	} else if err != nil {
		return passReq, errors2.APIError{
			Err:     err,
			Message: "Invalid JSON data.",
			Code:    http.StatusBadRequest,
		}
	}

	return passReq, nil
}

func (al *APIListener) handleListVaultValues(w http.ResponseWriter, req *http.Request) {
	items, err := al.vaultManager.List(req.Context(), req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(items))
}

func (al *APIListener) readIntParam(paramName string, req *http.Request) (int, error) {
	vars := mux.Vars(req)
	idStr, ok := vars[paramName]
	if !ok {
		return 0, nil
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("Non-numeric integer value provided: %s for param %s", idStr, paramName)
	}

	return id, nil
}

func (al *APIListener) handleReadVaultValue(w http.ResponseWriter, req *http.Request) {
	id, err := al.readIntParam(routeParamVaultValueID, req)
	if err != nil {
		al.jsonError(w, errors2.APIError{
			Err:  err,
			Code: http.StatusBadRequest,
		})
		return
	}
	if id == 0 {
		al.jsonError(w, errors2.APIError{
			Err:  fmt.Errorf("missing %q route param", routeParamVaultValueID),
			Code: http.StatusBadRequest,
		})
		return
	}

	curUser, err := al.getUserModelForAuth(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	storedValue, found, err := al.vaultManager.GetOne(req.Context(), id, curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if !found {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Cannot find a vault value by the provided id: %d", id))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleVaultStoreValue(w http.ResponseWriter, req *http.Request) {
	id, err := al.readIntParam(routeParamVaultValueID, req)
	if err != nil {
		al.jsonError(w, errors2.APIError{
			Err:  err,
			Code: http.StatusBadRequest,
		})
		return
	}

	curUser, err := al.getUserModelForAuth(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	var vaultKeyValue vault.InputValue
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err = dec.Decode(&vaultKeyValue)
	if err == io.EOF {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Missing body with json data.")
		return
	} else if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid JSON data.", err)
		return
	}

	storedValue, err := al.vaultManager.Store(req.Context(), int64(id), &vaultKeyValue, curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	status := http.StatusOK

	if id == 0 {
		w.WriteHeader(http.StatusCreated)
	}

	al.writeJSONResponse(w, status, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleVaultDeleteValue(w http.ResponseWriter, req *http.Request) {
	id, err := al.readIntParam(routeParamVaultValueID, req)
	if err != nil {
		al.jsonError(w, errors2.APIError{
			Err:  err,
			Code: http.StatusBadRequest,
		})
		return
	}
	if id == 0 {
		al.jsonError(w, errors2.APIError{
			Err:  fmt.Errorf("missing %q route param", routeParamVaultValueID),
			Code: http.StatusBadRequest,
		})
		return
	}

	curUser, err := al.getUserModelForAuth(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.vaultManager.Delete(req.Context(), id, curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) handleListScripts(w http.ResponseWriter, req *http.Request) {
	items, err := al.scriptManager.List(req.Context(), req)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(items))
}

func (al *APIListener) handleScriptCreate(w http.ResponseWriter, req *http.Request) {
	var scriptInput script.InputScript
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&scriptInput)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid JSON data.", err)
		return
	}

	curUsername := api.GetUser(req.Context(), al.Logger)
	if curUsername == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	storedValue, err := al.scriptManager.Create(req.Context(), &scriptInput, curUsername)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	al.writeJSONResponse(w, http.StatusCreated, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleScriptUpdate(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr, ok := vars[routeParamScriptValueID]
	if !ok {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Script ID is not provided")
		return
	}

	curUsername := api.GetUser(req.Context(), al.Logger)
	if curUsername == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var scriptInput script.InputScript
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&scriptInput)

	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid JSON data.", err)
		return
	}

	storedValue, err := al.scriptManager.Update(req.Context(), idStr, &scriptInput, curUsername)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(storedValue))
}

func (al *APIListener) handleReadScript(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr, ok := vars[routeParamScriptValueID]
	if !ok || idStr == "" {
		al.jsonError(w, errors2.APIError{
			Err:  errors.New("empty script id provided"),
			Code: http.StatusBadRequest,
		})
		return
	}

	foundScript, found, err := al.scriptManager.GetOne(req.Context(), idStr)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if !found {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Cannot find a script by the provided id: %s", idStr))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(foundScript))
}

func (al *APIListener) handleDeleteScript(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr, ok := vars[routeParamScriptValueID]
	if !ok || idStr == "" {
		al.jsonError(w, errors2.APIError{
			Err:  errors.New("empty script id provided"),
			Code: http.StatusBadRequest,
		})
		return
	}

	err := al.scriptManager.Delete(req.Context(), idStr)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
