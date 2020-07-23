package chserver

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func (al *APIListener) initRouter() {
	r := mux.NewRouter()
	sub := r.PathPrefix("/api/v1").Subrouter()
	sub.HandleFunc("/status", al.handleGetStatus).Methods(http.MethodGet)
	sub.HandleFunc("/sessions", al.handleGetSessions).Methods(http.MethodGet)
	sub.HandleFunc("/sessions/{session_id}/tunnels", al.handlePutSessionTunnel).Methods(http.MethodPut)
	sub.HandleFunc("/sessions/{session_id}/tunnels/{tunnel_id}", al.handleDeleteSessionTunnel).Methods(http.MethodDelete)

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
		al.Infof("error writing response: %s", err)
	}
}

func (al *APIListener) jsonErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	al.writeJSONResponse(w, statusCode, map[string]string{"error": err.Error()})
}

func (al *APIListener) handleGetStatus(w http.ResponseWriter, req *http.Request) {
	al.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"version":        chshare.BuildVersion,
		"sessions_count": al.sessionRepo.Count(),
	})
}

func (al *APIListener) handleGetSessions(w http.ResponseWriter, req *http.Request) {
	al.writeJSONResponse(w, http.StatusOK, al.sessionRepo.GetAll())
}

func (al *APIListener) handlePutSessionTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionID, exists := vars["session_id"]
	if !exists || sessionID == "" {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.Errorf("invalid session id supplied: %s", sessionID))
		return
	}

	session := al.sessionRepo.FindOne(sessionID)
	if session == nil {
		al.jsonErrorResponse(w, http.StatusNotFound, al.Errorf("session not found"))
		return
	}

	localAddr := req.URL.Query().Get("local")
	remoteAddr := req.URL.Query().Get("remote")
	remote, err := chshare.DecodeRemote(localAddr + ":" + remoteAddr)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.Errorf("invalid request: %s", err))
		return
	}

	// check user permissions
	if session.User != nil && !session.User.HasAccess(remote) {
		al.jsonErrorResponse(w, http.StatusForbidden, al.Errorf("access is not allowed for current session"))
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
		al.jsonErrorResponse(w, http.StatusConflict, al.Errorf("can't create tunnel: %s", err))
		return
	}
	response["tunnel_id"] = tunnelID

	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) handleDeleteSessionTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionID, exists := vars["session_id"]
	if !exists || sessionID == "" {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.Errorf("invalid session id supplied: %s", sessionID))
		return
	}

	session := al.sessionRepo.FindOne(sessionID)
	if session == nil {
		al.jsonErrorResponse(w, http.StatusNotFound, al.Errorf("session not found"))
		return
	}

	tunnelID, exists := vars["tunnel_id"]
	if !exists || tunnelID == "" {
		al.jsonErrorResponse(w, http.StatusBadRequest, al.Errorf("invalid session id supplied: %s", sessionID))
		return
	}

	// make next steps thread-safe
	session.Lock()
	defer session.Unlock()

	tunnel := session.FindTunnel(tunnelID)
	if tunnel == nil {
		al.jsonErrorResponse(w, http.StatusNotFound, al.Errorf("tunnel not found"))
		return
	}

	session.TerminateTunnel(tunnel)

	response := map[string]interface{}{"success": 1}
	al.writeJSONResponse(w, http.StatusOK, response)
}
