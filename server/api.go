package chserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
)

const (
	queryParamSort = "sort"
)

func (al *APIListener) wrapWithAuthMiddleware(f http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorized, username, err := al.lookupUser(r)
		if err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if !authorized || username == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="rportd-api"`)
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

	// add authorization middleware if needed
	if al.authorizationOn {
		_ = sub.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			route.HandlerFunc(al.wrapWithAuthMiddleware(route.GetHandler()))
			return nil
		})
	}

	// all routes defined below will not require authorization
	sub.HandleFunc("/login", al.handlePostLogin).Methods(http.MethodPost)
	sub.HandleFunc("/login", al.handleDeleteLogin).Methods(http.MethodDelete)

	// add max bytes middleware
	if al.authorizationOn {
		_ = sub.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			route.HandlerFunc(middleware.MaxBytes(route.GetHandler(), al.maxRequestBytes))
			return nil
		})
	}

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

func (al *APIListener) jsonErrorResponseWithErrCode(w http.ResponseWriter, statusCode int, errCode, msg string) {
	al.writeJSONResponse(w, statusCode, api.NewErrorPayloadWithCode(errCode, msg, ""))
}

func (al *APIListener) jsonErrorResponseWithDetail(w http.ResponseWriter, statusCode int, errCode, msg, detail string) {
	al.writeJSONResponse(w, statusCode, api.NewErrorPayloadWithCode(errCode, msg, detail))
}

func (al *APIListener) jsonErrorResponseWithError(w http.ResponseWriter, statusCode int, errCode, msg string, err error) {
	var detail string
	if err != nil {
		detail = err.Error()
	}
	al.writeJSONResponse(w, statusCode, api.NewErrorPayloadWithCode(errCode, msg, detail))
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
		"connect_url":    al.connectURL,
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
	sessionID, exists := vars["session_id"]
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
		Timeout:  al.checkPortTimeout,
	}
	resp := &comm.CheckPortResponse{}
	if err := comm.HandleSSHRequestJSON(conn, comm.RequestTypeCheckPort, req, resp); err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
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
	sessionID, exists := vars["session_id"]
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
