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
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
)

const (
	queryParamSort = "sort"

	routeParamSessionID = "session_id"
	routeParamJobID     = "job_id"

	ErrCodeMissingRouteVar = "ERR_CODE_MISSING_ROUTE_VAR"
	ErrCodeInvalidRequest  = "ERR_CODE_INVALID_REQUEST"
	ErrCodeAlreadyExist    = "ERR_CODE_ALREADY_EXIST"

	ErrCodeRunCmdDisabled = "ERR_CODE_RUN_CMD_DISABLED"
)

var validInputShell = []string{"cmd", "powershell"}

var generateNewJobID = func() string {
	return random.UUID4()
}

type JobProvider interface {
	GetByJID(sid, jid string) (*models.Job, error)
	GetSummariesBySID(sid string) ([]*models.JobSummary, error)
	SaveJob(job *models.Job) error
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
	sub.HandleFunc("/clients", al.handleGetClients).Methods(http.MethodGet)
	sub.HandleFunc("/clients", al.handlePostClients).Methods(http.MethodPost)
	sub.HandleFunc("/clients/{client_id}", al.handleDeleteClient).Methods(http.MethodDelete)

	// add authorization middleware if needed
	if al.IsAuthorizationOn() {
		_ = sub.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			route.HandlerFunc(al.wrapWithAuthMiddleware(route.GetHandler()))
			return nil
		})
	}

	// all routes defined below will not require authorization
	sub.HandleFunc("/login", al.handlePostLogin).Methods(http.MethodPost)
	sub.HandleFunc("/login", al.handleDeleteLogin).Methods(http.MethodDelete)

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
	curJob.PID = sshResp.Pid
	curJob.StartedAt = sshResp.StartedAt
	curJob.Status = models.JobStatusRunning

	if err := al.jobProvider.SaveJob(&curJob); err != nil {
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

func (al *APIListener) allowRunCommands(w http.ResponseWriter) bool {
	if al.jobProvider == nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusMethodNotAllowed, ErrCodeRunCmdDisabled, "Persistent storage required. A data dir or a database table is required to activate this feature.")
		return false
	}
	return true
}
