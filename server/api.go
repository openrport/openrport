package chserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/clientsauth"
	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

const (
	queryParamSort = "sort"

	routeParamClientID = "client_id"
	routeParamJobID    = "job_id"
	routeParamGroupID  = "group_id"

	ErrCodeMissingRouteVar = "ERR_CODE_MISSING_ROUTE_VAR"
	ErrCodeInvalidRequest  = "ERR_CODE_INVALID_REQUEST"
	ErrCodeAlreadyExist    = "ERR_CODE_ALREADY_EXIST"
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
	sub.HandleFunc("/me/ip", al.handleGetIP).Methods(http.MethodGet)
	sub.HandleFunc("/clients", al.handleGetClients).Methods(http.MethodGet)
	sub.HandleFunc("/clients/{client_id}/tunnels", al.handlePutClientTunnel).Methods(http.MethodPut)
	sub.HandleFunc("/clients/{client_id}/tunnels/{tunnel_id}", al.handleDeleteClientTunnel).Methods(http.MethodDelete)
	sub.HandleFunc("/clients/{client_id}/commands", al.handlePostCommand).Methods(http.MethodPost)
	sub.HandleFunc("/clients/{client_id}/commands", al.handleGetCommands).Methods(http.MethodGet)
	sub.HandleFunc("/clients/{client_id}/commands/{job_id}", al.handleGetCommand).Methods(http.MethodGet)
	sub.HandleFunc("/client-groups", al.handleGetClientGroups).Methods(http.MethodGet)
	sub.HandleFunc("/client-groups", al.handlePostClientGroups).Methods(http.MethodPost)
	sub.HandleFunc("/client-groups/{group_id}", al.handlePutClientGroup).Methods(http.MethodPut)
	sub.HandleFunc("/client-groups/{group_id}", al.handleGetClientGroup).Methods(http.MethodGet)
	sub.HandleFunc("/client-groups/{group_id}", al.handleDeleteClientGroup).Methods(http.MethodDelete)
	sub.HandleFunc("/commands", al.handlePostMultiClientCommand).Methods(http.MethodPost)
	sub.HandleFunc("/commands", al.handleGetMultiClientCommands).Methods(http.MethodGet)
	sub.HandleFunc("/commands/{job_id}", al.handleGetMultiClientCommand).Methods(http.MethodGet)
	sub.HandleFunc("/clients-auth", al.handleGetClientsAuth).Methods(http.MethodGet)
	sub.HandleFunc("/clients-auth", al.handlePostClientsAuth).Methods(http.MethodPost)
	sub.HandleFunc("/clients-auth/{client_auth_id}", al.handleDeleteClientAuth).Methods(http.MethodDelete)

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

	// web sockets
	// common auth middleware is not used due to JS issue https://stackoverflow.com/questions/22383089/is-it-possible-to-use-bearer-authentication-for-websocket-upgrade-requests
	sub.HandleFunc("/ws/commands", al.wsAuth(http.HandlerFunc(al.handleCommandsWS))).Methods(http.MethodGet)

	// only for test purpose
	// TODO: remove
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

	response := api.NewSuccessPayload(map[string]interface{}{
		"version":              chshare.BuildVersion,
		"clients_connected":    countActive,
		"clients_disconnected": countDisconnected,
		"fingerprint":          al.fingerprint,
		"connect_url":          al.config.Server.URL,
		"clients_auth_source":  al.clientAuthProvider.Source(),
		"clients_auth_mode":    al.getClientsAuthMode(),
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

const (
	URISchemeMaxLength = 15

	ErrCodeLocalPortInUse        = "ERR_CODE_LOCAL_PORT_IN_USE"
	ErrCodeRemotePortNotOpen     = "ERR_CODE_REMOTE_PORT_NOT_OPEN"
	ErrCodeTunnelExist           = "ERR_CODE_TUNNEL_EXIST"
	ErrCodeTunnelToPortExist     = "ERR_CODE_TUNNEL_TO_PORT_EXIST"
	ErrCodeURISchemeLengthExceed = "ERR_CODE_URI_SCHEME_LENGTH_EXCEED"
	ErrCodeInvalidACL            = "ERR_CODE_INVALID_ACL"
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
		if t.Remote.Remote() == remote.Remote() {
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
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "user not found")
		return
	}

	me := struct {
		User   string   `json:"user"`
		Groups []string `json:"groups"`
	}{
		User:   user.Username,
		Groups: user.Groups,
	}
	response := api.NewSuccessPayload(me)
	al.writeJSONResponse(w, http.StatusOK, response)
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

	client, err := al.clientService.GetActiveByID(cid)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "", fmt.Sprintf("Failed to find an active client with id=%q.", cid), err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Active client with id=%q not found.", cid))
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
		ClientID:   cid,
		ClientName: client.Name,
		Command:    reqBody.Command,
		Shell:      reqBody.Shell,
		CreatedBy:  api.GetUser(req.Context(), al.Logger),
		TimeoutSec: reqBody.TimeoutSec,
		Result:     nil,
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

	al.Debugf("Job[id=%q] created to execute remote command on client with id=%q: %q.", curJob.JID, cid, reqBody.Command)
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
	Shell               string   `json:"shell"`
	TimeoutSec          int      `json:"timeout_sec"`
	ExecuteConcurrently bool     `json:"execute_concurrently"`
	AbortOnError        *bool    `json:"abort_on_error"` // pointer is used because it's default value is true. Otherwise it would be more difficult to check whether this field is missing or not
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
			go al.createAndRunJob(job.JID, job.Command, job.Shell, job.CreatedBy, job.TimeoutSec, client)
		} else {
			success := al.createAndRunJob(job.JID, job.Command, job.Shell, job.CreatedBy, job.TimeoutSec, client)
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

func (al *APIListener) createAndRunJob(jid, cmd, shell, createdBy string, timeoutSec int, client *clients.Client) bool {
	// send the command to the client
	curJob := models.Job{
		JobSummary: models.JobSummary{
			JID: generateNewJobID(),
		},
		StartedAt:  time.Now(),
		ClientID:   client.ID,
		ClientName: client.Name,
		Command:    cmd,
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
	inboundMsg := multiClientCmdRequest{}
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

	createdBy := api.GetUser(req.Context(), al.Logger)
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
			Shell:      inboundMsg.Shell,
			TimeoutSec: inboundMsg.TimeoutSec,
			Concurrent: inboundMsg.ExecuteConcurrently,
			AbortOnErr: abortOnErr,
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
				go al.createAndRunJobWS(uiConnTS, &jid, curJID, multiJob.Command, multiJob.Shell, createdBy, multiJob.TimeoutSec, client)
			} else {
				success := al.createAndRunJobWS(uiConnTS, &jid, curJID, multiJob.Command, multiJob.Shell, createdBy, multiJob.TimeoutSec, client)
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
		al.createAndRunJobWS(uiConnTS, nil, jid, inboundMsg.Command, inboundMsg.Shell, createdBy, inboundMsg.TimeoutSec, orderedClients[0])
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

func (al *APIListener) createAndRunJobWS(uiConnTS *ws.ConcurrentWebSocket, multiJobID *string, jid, cmd, shell, createdBy string, timeoutSec int, client *clients.Client) bool {
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
