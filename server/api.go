package chserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/hgroups"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

const (
	queryParamSort = "sort"

	routeParamSessionID = "session_id"
	routeParamJobID     = "job_id"
	routeParamGroupID   = "group_id"

	ErrCodeMissingRouteVar = "ERR_CODE_MISSING_ROUTE_VAR"
	ErrCodeInvalidRequest  = "ERR_CODE_INVALID_REQUEST"
	ErrCodeAlreadyExist    = "ERR_CODE_ALREADY_EXIST"

	ErrCodeRunCmdDisabled = "ERR_CODE_RUN_CMD_DISABLED"
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
	GetByJID(sid, jid string) (*models.Job, error)
	GetSummariesBySID(sid string) ([]*models.JobSummary, error)
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
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if !authorized || username == "" {
			al.jsonErrorResponse(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}

		newCtx := api.WithUser(r.Context(), username)
		f.ServeHTTP(w, r.WithContext(newCtx))
	}
}

func (al *APIListener) initRouter() {
	r := mux.NewRouter()
	sub := r.PathPrefix("/api/v1").Subrouter()
	sub.HandleFunc("/login", al.handleGetLogin).Methods(http.MethodGet)
	sub.HandleFunc("/status", al.handleGetStatus).Methods(http.MethodGet)
	sub.HandleFunc("/me", al.handleGetMe).Methods(http.MethodGet)
	sub.HandleFunc("/sessions", al.handleGetSessions).Methods(http.MethodGet)
	sub.HandleFunc("/sessions/{session_id}/tunnels", al.handlePutSessionTunnel).Methods(http.MethodPut)
	sub.HandleFunc("/sessions/{session_id}/tunnels/{tunnel_id}", al.handleDeleteSessionTunnel).Methods(http.MethodDelete)
	sub.HandleFunc("/sessions/{session_id}/commands", al.handlePostCommand).Methods(http.MethodPost)
	sub.HandleFunc("/sessions/{session_id}/commands", al.handleGetCommands).Methods(http.MethodGet)
	sub.HandleFunc("/sessions/{session_id}/commands/{job_id}", al.handleGetCommand).Methods(http.MethodGet)
	sub.HandleFunc("/host-groups", al.handleGetHostGroups).Methods(http.MethodGet)
	sub.HandleFunc("/host-groups", al.handlePostHostGroups).Methods(http.MethodPost)
	sub.HandleFunc("/host-groups/{group_id}", al.handlePutHostGroup).Methods(http.MethodPut)
	sub.HandleFunc("/host-groups/{group_id}", al.handleGetHostGroup).Methods(http.MethodGet)
	sub.HandleFunc("/host-groups/{group_id}", al.handleDeleteHostGroup).Methods(http.MethodDelete)
	sub.HandleFunc("/commands/ws", al.handleCommandsWS).Methods(http.MethodGet)
	sub.HandleFunc("/commands", al.handlePostMultiClientCommand).Methods(http.MethodPost)
	sub.HandleFunc("/commands", al.handleGetMultiClientCommands).Methods(http.MethodGet)
	sub.HandleFunc("/commands/{job_id}", al.handleGetMultiClientCommand).Methods(http.MethodGet)
	sub.HandleFunc("/clients", al.handleGetClients).Methods(http.MethodGet)
	sub.HandleFunc("/clients", al.handlePostClients).Methods(http.MethodPost)
	sub.HandleFunc("/clients/{client_id}", al.handleDeleteClient).Methods(http.MethodDelete)

	// add authorization middleware
	if !al.insecureForTests {
		_ = sub.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			route.HandlerFunc(al.wrapWithAuthMiddleware(route.GetHandler()))
			return nil
		})
	}

	// all routes defined below will not require authorization
	sub.HandleFunc("/login", al.handlePostLogin).Methods(http.MethodPost)
	sub.HandleFunc("/login", al.handleDeleteLogin).Methods(http.MethodDelete)

	// only for test purpose
	// TODO: remove
	sub.HandleFunc("/test/commands/ws", al.handleCommandsWS).Methods(http.MethodGet)
	sub.HandleFunc("/test/commands/ui", al.home)

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

func (al *APIListener) handleGetLogin(w http.ResponseWriter, req *http.Request) {
	lifetime, err := parseTokenLifetime(req)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	tokenStr, err := al.createAuthToken(lifetime, api.GetUser(req.Context(), al.Logger))
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := api.NewSuccessPayload(map[string]string{"token": tokenStr})
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handlePostLogin(w http.ResponseWriter, req *http.Request) {
	lifetime, err := parseTokenLifetime(req)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	user, pwd, err := parseLoginPostRequestBody(req)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, fmt.Errorf("can't parse request body: %s", err))
		return
	}

	authorized, err := al.validateCredentials(user, pwd)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("can't validate credentials: %v", err))
		return
	}
	if !authorized {
		al.jsonErrorResponse(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return
	}

	tokenStr, err := al.createAuthToken(lifetime, user)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := api.NewSuccessPayload(map[string]string{"token": tokenStr})
	al.writeJSONResponse(w, http.StatusOK, response)
}

func parseLoginPostRequestBody(req *http.Request) (string, string, error) {
	reqContentType := req.Header.Get("Content-Type")
	if reqContentType == "application/x-www-form-urlencoded" {
		err := req.ParseForm()
		if err != nil {
			return "", "", err
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
			return "", "", err
		}
		return params.Username, params.Password, nil
	}
	return "", "", fmt.Errorf("unsupported content type")
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

func (al *APIListener) handleDeleteLogin(w http.ResponseWriter, req *http.Request) {
	token, tokenProvided := getBearerToken(req)
	if token == "" || !tokenProvided {
		al.jsonErrorResponse(w, http.StatusBadRequest, fmt.Errorf("authorization Bearer token required"))
		return
	}

	valid, _, apiSession, err := al.validateBearerToken(token)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if !valid {
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

func (al *APIListener) handleGetStatus(w http.ResponseWriter, req *http.Request) {
	count, err := al.sessionService.Count()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := api.NewSuccessPayload(map[string]interface{}{
		"version":        chshare.BuildVersion,
		"sessions_count": count,
		"fingerprint":    al.fingerprint,
		"connect_url":    al.config.Server.URL,
	})
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleGetSessions(w http.ResponseWriter, req *http.Request) {
	sortFunc, desc, err := getCorrespondingSortFunc(req.URL.Query().Get(queryParamSort))
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	clientSessions, err := al.sessionService.GetAll()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	sortFunc(clientSessions, desc)

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(clientSessions))
}

func getCorrespondingSortFunc(sortStr string) (sortFunc func(a []*sessions.ClientSession, desc bool), desc bool, err error) {
	var sortField string
	if strings.HasPrefix(sortStr, "-") {
		desc = true
		sortField = sortStr[1:]
	} else {
		sortField = sortStr
	}

	switch sortField {
	case "":
		sortFunc = sessions.SortByID
	case "id":
		sortFunc = sessions.SortByID
	case "name":
		sortFunc = sessions.SortByName
	case "os":
		sortFunc = sessions.SortByOS
	case "hostname":
		sortFunc = sessions.SortByHostname
	case "version":
		sortFunc = sessions.SortByVersion
	default:
		err = fmt.Errorf("incorrect format of %q query param", queryParamSort)
	}
	return
}

const (
	URISchemeMaxLength = 15

	ErrCodePortInUse             = "ERR_CODE_PORT_IN_USE"
	ErrCodePortNotOpen           = "ERR_CODE_PORT_NOT_OPEN"
	ErrCodeTunnelExist           = "ERR_CODE_TUNNEL_EXIST"
	ErrCodeTunnelToPortExist     = "ERR_CODE_TUNNEL_TO_PORT_EXIST"
	ErrCodeURISchemeLengthExceed = "ERR_CODE_URI_SCHEME_LENGTH_EXCEED"
)

func (al *APIListener) handlePutSessionTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionID, exists := vars[routeParamSessionID]
	if !exists || sessionID == "" {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.FormatError("invalid session id supplied: %s", sessionID))
		return
	}

	session, err := al.sessionService.GetActiveByID(sessionID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if session == nil {
		al.jsonErrorResponse(w, http.StatusNotFound, al.FormatError("session not found"))
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
		al.jsonErrorResponse(w, http.StatusBadRequest, al.FormatError("invalid request: %s", err))
		return
	}

	aclStr := req.URL.Query().Get("acl")
	if _, err = sessions.ParseTunnelACL(aclStr); err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.FormatError("invalid request: %s", err))
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

	if existing := session.FindTunnelByRemote(remote); existing != nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeTunnelExist, "Tunnel already exist.")
		return
	}

	for _, t := range session.Tunnels {
		if t.Remote.Remote() == remote.Remote() {
			al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeTunnelToPortExist, fmt.Sprintf("Tunnel to port %s already exist.", remote.RemotePort))
			return
		}
	}

	if checkPortStr := req.URL.Query().Get("check_port"); checkPortStr != "0" {
		if !al.checkRemotePort(w, *remote, session.Connection) {
			return
		}
	}

	// make next steps thread-safe
	session.Lock()
	defer session.Unlock()

	if remote.IsLocalSpecified() && !al.checkLocalPort(w, remote.LocalPort) {
		return
	}

	tunnels, err := al.sessionService.StartSessionTunnels(session, []*chshare.Remote{remote})
	if err != nil {
		al.jsonErrorResponse(w, http.StatusConflict, al.FormatError("can't create tunnel: %s", err))
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
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodePortInUse, fmt.Sprintf("Port %d already in use.", lport))
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
			al.jsonErrorResponseWithTitle(w, http.StatusConflict, err.Error())
		} else {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		}
		return false
	}

	if !resp.Open {
		al.jsonErrorResponseWithDetail(
			w,
			http.StatusBadRequest,
			ErrCodePortNotOpen,
			fmt.Sprintf("Port %s is not in listening state.", remote.RemotePort),
			resp.ErrMsg,
		)
		return false
	}

	return true
}

func (al *APIListener) handleDeleteSessionTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionID, exists := vars[routeParamSessionID]
	if !exists || sessionID == "" {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.FormatError("invalid session id supplied: %s", sessionID))
		return
	}

	session, err := al.sessionService.GetActiveByID(sessionID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if session == nil {
		al.jsonErrorResponse(w, http.StatusNotFound, al.FormatError("session not found"))
		return
	}

	tunnelID, exists := vars["tunnel_id"]
	if !exists || tunnelID == "" {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.FormatError("invalid tunnel id supplied: %s", sessionID))
		return
	}

	// make next steps thread-safe
	session.Lock()
	defer session.Unlock()

	tunnel := session.FindTunnel(tunnelID)
	if tunnel == nil {
		al.jsonErrorResponse(w, http.StatusNotFound, al.FormatError("tunnel not found"))
		return
	}

	session.TerminateTunnel(tunnel)

	w.WriteHeader(http.StatusNoContent)
}

// handleGetMe returns the currently logged in user and the groups the user belongs to.
func (al *APIListener) handleGetMe(w http.ResponseWriter, req *http.Request) {
	curUsername := api.GetUser(req.Context(), al.Logger)
	if curUsername == "" {
		al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(nil))
		return
	}

	user, err := al.userSrv.GetByUsername(curUsername)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if user == nil {
		al.jsonErrorResponse(w, http.StatusNotFound, al.FormatError("user not found"))
		return
	}

	me := struct {
		User   string   `json:"user"`
		Groups []string `json:"groups,omitempty"`
	}{
		User:   user.Username,
		Groups: user.Groups,
	}
	response := api.NewSuccessPayload(me)
	al.writeJSONResponse(w, http.StatusOK, response)
}

const (
	MinCredentialsLength = 3

	ErrCodeClientAuthDisabled     = "ERR_CODE_CLIENT_AUTH_DISABLED"
	ErrCodeClientAuthSingleClient = "ERR_CODE_CLIENT_AUTH_SINGLE"
	ErrCodeClientAuthRO           = "ERR_CODE_CLIENT_AUTH_RO"

	ErrCodeClientHasSession = "ERR_CODE_CLIENT_HAS_SESSION"
	ErrCodeClientNotFound   = "ERR_CODE_CLIENT_NOT_FOUND"
)

func (al *APIListener) handleGetClients(w http.ResponseWriter, req *http.Request) {
	rClients, err := al.clientProvider.GetAll()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	clients.SortByID(rClients, false)

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(rClients))
}

func (al *APIListener) handlePostClients(w http.ResponseWriter, req *http.Request) {
	if !al.allowClientAuthWrite(w) {
		return
	}

	var newClient clients.Client
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

	added, err := al.clientProvider.Add(&newClient)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if !added {
		al.jsonErrorResponseWithDetail(w, http.StatusConflict, ErrCodeAlreadyExist, fmt.Sprintf("Client with ID %q already exist.", newClient.ID), "")
		return
	}

	al.Infof("Client %q created.", newClient.ID)

	w.WriteHeader(http.StatusCreated)
}

func (al *APIListener) handleDeleteClient(w http.ResponseWriter, req *http.Request) {
	if !al.allowClientAuthWrite(w) {
		return
	}

	vars := mux.Vars(req)
	clientID := vars["client_id"]
	if clientID == "" {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeMissingRouteVar, "Missing 'client_id' route param.")
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

	existing, err := al.clientProvider.Get(clientID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if existing == nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeClientNotFound, fmt.Sprintf("Client with ID=%q not found.", clientID))
		return
	}

	allSessions := al.sessionService.GetAllByClientID(clientID)
	if !force && len(allSessions) > 0 {
		al.jsonErrorResponseWithErrCode(w, http.StatusConflict, ErrCodeClientHasSession, fmt.Sprintf("Client expected to have no active or disconnected session(s), got %d.", len(allSessions)))
		return
	}

	for _, s := range allSessions {
		if err := al.sessionService.ForceDelete(s); err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
	}

	err = al.clientProvider.Delete(clientID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	al.Infof("Client %q deleted.", clientID)

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) allowClientAuthWrite(w http.ResponseWriter) bool {
	if !al.clientProvider.IsWriteable() {
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
	if !al.allowRunCommands(w) {
		return
	}

	vars := mux.Vars(req)
	sid := vars[routeParamSessionID]
	if sid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamSessionID))
		return
	}

	reqBody := struct {
		Command    string `json:"command"`
		Shell      string `json:"shell"`
		TimeoutSec int    `json:"timeout_sec"`
	}{}
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

	session, err := al.sessionService.GetActiveByID(sid)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to find an active client session with id=%q.", sid), err)
		return
	}
	if session == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Active session with id=%q not found.", sid))
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
		SID:        sid,
		Command:    reqBody.Command,
		Shell:      reqBody.Shell,
		CreatedBy:  api.GetUser(req.Context(), al.Logger),
		TimeoutSec: reqBody.TimeoutSec,
		Result:     nil,
	}
	sshResp := &comm.RunCmdResponse{}
	err = comm.SendRequestAndGetResponse(session.Connection, comm.RequestTypeRunCmd, curJob, sshResp)
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

	al.Debugf("Job[id=%q] created to execute remote command on client with sessionID=%q: %q.", curJob.JID, sid, reqBody.Command)
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
	if !al.allowRunCommands(w) {
		return
	}

	vars := mux.Vars(req)
	sid := vars[routeParamSessionID]
	if sid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamSessionID))
		return
	}

	res, err := al.jobProvider.GetSummariesBySID(sid)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to get client jobs: session_id=%q.", sid), err)
		return
	}

	jobs.SortByFinishedAt(res, true)
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(res))
}

func (al *APIListener) handleGetCommand(w http.ResponseWriter, req *http.Request) {
	if !al.allowRunCommands(w) {
		return
	}

	vars := mux.Vars(req)
	sid := vars[routeParamSessionID]
	if sid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamSessionID))
		return
	}
	jid := vars[routeParamJobID]
	if jid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamJobID))
		return
	}

	job, err := al.jobProvider.GetByJID(sid, jid)
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

func (al *APIListener) handlePostMultiClientCommand(w http.ResponseWriter, req *http.Request) {
	if !al.allowRunCommands(w) {
		return
	}

	reqBody := struct {
		ClientIDs           []string `json:"client_ids"`
		Command             string   `json:"command"`
		Shell               string   `json:"shell"`
		TimeoutSec          int      `json:"timeout_sec"`
		ExecuteConcurrently bool     `json:"execute_concurrently"`
		AbortOnError        bool     `json:"abort_on_error"`
	}{}
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

	minClients := 2
	if len(reqBody.ClientIDs) < minClients {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("At least %d clients should be specified.", minClients))
		return
	}

	clientsConn := make(map[string]ssh.Conn)
	for _, sid := range reqBody.ClientIDs {
		session, err := al.sessionService.GetByID(sid)
		if err != nil {
			al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to find a client session with id=%q.", sid), err)
			return
		}
		if session == nil {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Session with id=%q not found.", sid))
			return
		}
		if session.Disconnected != nil {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Session with id=%q is not active.", sid))
			return
		}
		clientsConn[sid] = session.Connection
	}

	multiJob := &models.MultiJob{
		MultiJobSummary: models.MultiJobSummary{
			JID:       generateNewJobID(),
			StartedAt: time.Now(),
			CreatedBy: api.GetUser(req.Context(), al.Logger),
		},
		ClientIDs:  reqBody.ClientIDs,
		Command:    reqBody.Command,
		Shell:      reqBody.Shell,
		TimeoutSec: reqBody.TimeoutSec,
		Concurrent: reqBody.ExecuteConcurrently,
		AbortOnErr: reqBody.AbortOnError,
	}
	if err := al.jobProvider.SaveMultiJob(multiJob); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to persist a new multi-client job.", err)
		return
	}

	resp := newJobResponse{
		JID: multiJob.JID,
	}
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(resp))

	al.Debugf("Multi-client Job[id=%q] created to execute remote command on clients %s: %q.", multiJob.JID, reqBody.ClientIDs, reqBody.Command)

	go al.executeMultiClientJob(multiJob, clientsConn)
}

func (al *APIListener) executeMultiClientJob(job *models.MultiJob, clientsConn map[string]ssh.Conn) {
	for _, sid := range job.ClientIDs {
		if job.Concurrent {
			go al.createAndRunJob(job.JID, sid, job.Command, job.Shell, job.CreatedBy, job.TimeoutSec, clientsConn[sid])
		} else {
			success := al.createAndRunJob(job.JID, sid, job.Command, job.Shell, job.CreatedBy, job.TimeoutSec, clientsConn[sid])
			if !success && job.AbortOnErr {
				break
			}
		}
	}
	if al.testDone != nil {
		al.testDone <- true
	}
}

func (al *APIListener) createAndRunJob(jid, sid, cmd, shell, createdBy string, timeoutSec int, conn ssh.Conn) bool {
	// send the command to the client
	curJob := models.Job{
		JobSummary: models.JobSummary{
			JID: generateNewJobID(),
		},
		StartedAt:  time.Now(),
		SID:        sid,
		Command:    cmd,
		Shell:      shell,
		CreatedBy:  createdBy,
		TimeoutSec: timeoutSec,
		MultiJobID: &jid,
	}
	sshResp := &comm.RunCmdResponse{}
	err := comm.SendRequestAndGetResponse(conn, comm.RequestTypeRunCmd, curJob, sshResp)
	// return an error after saving the job
	if err != nil {
		// failure, set fields to mark it as failed
		al.Errorf("multi_client_id=%q, sid=%q, Error on execute remote command: %v", *curJob.MultiJobID, curJob.SID, err)
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
		al.Errorf("multi_client_id=%q, sid=%q, Failed to persist a child job: %v", *curJob.MultiJobID, curJob.SID, dbErr)
	}

	return err == nil
}

func (al *APIListener) handleCommandsWS(w http.ResponseWriter, req *http.Request) {
	if !al.allowRunCommands(w) {
		return
	}

	uiConn, err := apiUpgrader.Upgrade(w, req, nil)
	if err != nil {
		al.Errorf("Failed to establish WS connection: %v", err)
		return
	}
	uiConnTS := ws.NewConcurrentWebSocket(uiConn, al.Logger)
	inboundMsg := struct {
		ClientIDs           []string `json:"client_ids"`
		Command             string   `json:"command"`
		Shell               string   `json:"shell"`
		TimeoutSec          int      `json:"timeout_sec"`
		ExecuteConcurrently bool     `json:"execute_concurrently"`
		AbortOnError        bool     `json:"abort_on_error"`
	}{}
	err = uiConnTS.ReadJSON(&inboundMsg)
	if err == io.EOF { // is handled separately to return an informative error message
		uiConnTS.WriteError("Inbound message should contain non empty json object with command data.", nil)
		return
	} else if err != nil {
		uiConnTS.WriteError("Invalid JSON data.", err)
		return
	}

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

	if len(inboundMsg.ClientIDs) < 1 {
		uiConnTS.WriteError("'client_ids' field should contain at least one client ID", nil)
		return
	}

	clientsConn := make(map[string]ssh.Conn)
	for _, sid := range inboundMsg.ClientIDs {
		session, err := al.sessionService.GetByID(sid)
		if err != nil {
			uiConnTS.WriteError(fmt.Sprintf("Failed to find a client session with id=%q.", sid), err)
			return
		}
		if session == nil {
			uiConnTS.WriteError(fmt.Sprintf("Session with id=%q not found.", sid), nil)
			return
		}
		if session.Disconnected != nil {
			uiConnTS.WriteError(fmt.Sprintf("Session with id=%q is not active.", sid), nil)
			return
		}
		clientsConn[sid] = session.Connection
	}

	jid := generateNewJobID()
	al.Server.uiJobWebSockets.Set(jid, uiConnTS)
	defer al.Server.uiJobWebSockets.Delete(jid)

	createdBy := api.GetUser(req.Context(), al.Logger)
	if len(inboundMsg.ClientIDs) > 1 {
		multiJob := &models.MultiJob{
			MultiJobSummary: models.MultiJobSummary{
				JID:       jid,
				StartedAt: time.Now(),
				CreatedBy: createdBy,
			},
			ClientIDs:  inboundMsg.ClientIDs,
			Command:    inboundMsg.Command,
			Shell:      inboundMsg.Shell,
			TimeoutSec: inboundMsg.TimeoutSec,
			Concurrent: inboundMsg.ExecuteConcurrently,
			AbortOnErr: inboundMsg.AbortOnError,
		}
		if err := al.jobProvider.SaveMultiJob(multiJob); err != nil {
			uiConnTS.WriteError("Failed to persist a new multi-client job.", err)
			return
		}

		al.Debugf("Multi-client Job[id=%q] created to execute remote command on clients %s: %q.", multiJob.JID, inboundMsg.ClientIDs, inboundMsg.Command)
		uiConnTS.SetWritesBeforeClose(len(multiJob.ClientIDs))
		for _, sid := range multiJob.ClientIDs {
			curJID := generateNewJobID()
			if multiJob.Concurrent {
				go al.createAndRunJobWS(uiConnTS, &jid, curJID, sid, multiJob.Command, multiJob.Shell, createdBy, multiJob.TimeoutSec, clientsConn[sid])
			} else {
				success := al.createAndRunJobWS(uiConnTS, &jid, curJID, sid, multiJob.Command, multiJob.Shell, createdBy, multiJob.TimeoutSec, clientsConn[sid])
				if !success && multiJob.AbortOnErr {
					uiConnTS.Close()
					return
				}
				// TODO: wait until job will be finished
			}
		}
	} else {
		al.createAndRunJobWS(uiConnTS, nil, jid, inboundMsg.ClientIDs[0], inboundMsg.Command, inboundMsg.Shell, createdBy, inboundMsg.TimeoutSec, clientsConn[inboundMsg.ClientIDs[0]])
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

func (al *APIListener) createAndRunJobWS(uiConnTS *ws.ConcurrentWebSocket, multiJobID *string, jid, sid, cmd, shell, createdBy string, timeoutSec int, clientConn ssh.Conn) bool {
	curJob := models.Job{
		JobSummary: models.JobSummary{
			JID: jid,
		},
		StartedAt:  time.Now(),
		SID:        sid,
		Command:    cmd,
		Shell:      shell,
		CreatedBy:  createdBy,
		TimeoutSec: timeoutSec,
		MultiJobID: multiJobID,
	}
	logPrefix := curJob.LogPrefix()

	// send the command to the client
	sshResp := &comm.RunCmdResponse{}
	err := comm.SendRequestAndGetResponse(clientConn, comm.RequestTypeRunCmd, curJob, sshResp)
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
	if !al.allowRunCommands(w) {
		return
	}

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
	if !al.allowRunCommands(w) {
		return
	}

	res, err := al.jobProvider.GetAllMultiJobSummaries()
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to get multi-client jobs.", err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(res))
}

func (al *APIListener) allowRunCommands(w http.ResponseWriter) bool {
	if al.jobProvider == nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusMethodNotAllowed, ErrCodeRunCmdDisabled, "Persistent storage required. A data dir or a database table is required to activate this feature.")
		return false
	}
	return true
}

func (al *APIListener) handlePostHostGroups(w http.ResponseWriter, req *http.Request) {
	var group hgroups.HostGroup
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

	if err := validateInputHostGroup(group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid host group.", err)
		return
	}

	if err := al.hostGroupProvider.Create(req.Context(), &group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to persist a new host group.", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	al.Debugf("Host Group [id=%q] created.", group.ID)
}

func (al *APIListener) handlePutHostGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars[routeParamGroupID]
	if id == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamGroupID))
		return
	}

	var group hgroups.HostGroup
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

	if err := validateInputHostGroup(group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "", "Invalid host group.", err)
		return
	}

	if err := al.hostGroupProvider.Update(req.Context(), &group); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to persist host group.", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("Host Group [id=%q] updated.", group.ID)
}

func validateInputHostGroup(group hgroups.HostGroup) error {
	if strings.TrimSpace(group.ID) == "" {
		return errors.New("ID cannot be empty")
	}
	return nil
}

func (al *APIListener) handleGetHostGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars[routeParamGroupID]
	if id == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamGroupID))
		return
	}

	group, err := al.hostGroupProvider.Get(req.Context(), id)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to find host group[id=%q].", id), err)
		return
	}
	if group == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Host Group[id=%q] not found.", id))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(group))
}

func (al *APIListener) handleGetHostGroups(w http.ResponseWriter, req *http.Request) {
	res, err := al.hostGroupProvider.GetAll(req.Context())
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", "Failed to get host groups.", err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(res))
}

func (al *APIListener) handleDeleteHostGroup(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars[routeParamGroupID]
	if id == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamGroupID))
		return
	}

	err := al.hostGroupProvider.Delete(req.Context(), id)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to delete host group[id=%q].", id), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("Host Group [id=%q] deleted.", id)
}
