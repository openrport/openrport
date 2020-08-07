package chserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func (al *APIListener) wrapWithAuthMiddleware(f http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok := al.handleAuthorization(w, r)
		if !ok {
			return
		}
		f.ServeHTTP(w, r)
	}
}

func (al *APIListener) initRouter() {
	r := mux.NewRouter()
	sub := r.PathPrefix("/api/v1").Subrouter()
	sub.HandleFunc("/login", al.handleGetLogin).Methods(http.MethodGet)
	sub.HandleFunc("/status", al.handleGetStatus).Methods(http.MethodGet)
	sub.HandleFunc("/sessions", al.handleGetSessions).Methods(http.MethodGet)
	sub.HandleFunc("/sessions/{session_id}/tunnels", al.handlePutSessionTunnel).Methods(http.MethodPut)
	sub.HandleFunc("/sessions/{session_id}/tunnels/{tunnel_id}", al.handleDeleteSessionTunnel).Methods(http.MethodDelete)

	// add authorization middleware
	_ = sub.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		route.HandlerFunc(al.wrapWithAuthMiddleware(route.GetHandler()))
		return nil
	})
	// all routes defined below will not require authorization

	sub.HandleFunc("/login", al.handlePostLogin).Methods(http.MethodPost)
	sub.HandleFunc("/login", al.handleDeleteLogin).Methods(http.MethodDelete)

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
	al.writeJSONResponse(w, statusCode, map[string]string{"error": err.Error()})
}

func (al *APIListener) handleGetLogin(w http.ResponseWriter, req *http.Request) {
	lifetime, err := parseTokenLifetime(req)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	tokenStr, err := al.createAuthToken(lifetime)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, map[string]string{"token": tokenStr})
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

	if !al.validateUserPasswordPair(user, pwd) {
		al.jsonErrorResponse(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return
	}

	tokenStr, err := al.createAuthToken(lifetime)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	al.writeJSONResponse(w, http.StatusOK, map[string]string{"token": tokenStr})
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

	valid, apiSession, err := al.validateBearerToken(token)
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

	response := map[string]interface{}{"success": 1}
	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleGetStatus(w http.ResponseWriter, req *http.Request) {
	count, err := al.sessionRepo.Count()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	al.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"version":        chshare.BuildVersion,
		"sessions_count": count,
	})
}

func (al *APIListener) handleGetSessions(w http.ResponseWriter, req *http.Request) {
	clientSessions, err := al.sessionRepo.GetAll()
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	al.writeJSONResponse(w, http.StatusOK, clientSessions)
}

func (al *APIListener) handlePutSessionTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionID, exists := vars["session_id"]
	if !exists || sessionID == "" {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.FormatError("invalid session id supplied: %s", sessionID))
		return
	}

	session, err := al.sessionRepo.FindOne(sessionID)
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
	remote, err := chshare.DecodeRemote(localAddr + ":" + remoteAddr)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.FormatError("invalid request: %s", err))
		return
	}

	response := map[string]interface{}{"success": 1}

	// make next steps thread-safe
	session.Lock()
	defer session.Unlock()

	// check if such remote already exists
	if session.HasRemote(remote) {
		response["msg"] = "requested tunnel already exists"
		al.writeJSONResponse(w, http.StatusNoContent, response)
		return
	}

	tunnelID, err := session.StartRemoteTunnel(remote)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusConflict, al.FormatError("can't create tunnel: %s", err))
		return
	}
	response["tunnel_id"] = tunnelID

	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleDeleteSessionTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionID, exists := vars["session_id"]
	if !exists || sessionID == "" {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.FormatError("invalid session id supplied: %s", sessionID))
		return
	}

	session, err := al.sessionRepo.FindOne(sessionID)
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

	response := map[string]interface{}{"success": 1}
	al.writeJSONResponse(w, http.StatusOK, response)
}
