package chserver

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func NewAPIRouter(s *Server) *mux.Router {
	r := mux.NewRouter()
	sub := r.PathPrefix("/api/v1").Subrouter()
	sub.HandleFunc("/status", s.handleGetStatus).Methods(http.MethodGet)
	sub.HandleFunc("/sessions", s.handleGetSessions).Methods(http.MethodGet)
	sub.HandleFunc("/sessions/{id}/tunnels", s.handlePutSessionTunnel).Methods(http.MethodPut)
	return r
}

func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	b, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	if _, err := w.Write(b); err != nil {
		s.Infof("error writing response: %s", err)
	}
}

func (s *Server) jsonErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	s.writeJSONResponse(w, statusCode, map[string]string{"error": err.Error()})
}

func (s *Server) handleGetStatus(w http.ResponseWriter, req *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"version":        chshare.BuildVersion,
		"sessions_count": len(s.sessions),
	})
}

func (s *Server) handleGetSessions(w http.ResponseWriter, req *http.Request) {
	var result = make([]*ClientSession, 0)
	for _, c := range s.sessions {
		result = append(result, c)
	}
	s.writeJSONResponse(w, http.StatusOK, result)
}

func (s *Server) handlePutSessionTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionID, exists := vars["id"]
	if !exists || sessionID == "" {
		s.jsonErrorResponse(w, http.StatusBadRequest, s.Errorf("invalid session id supplied: %s", sessionID))
		return
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		s.jsonErrorResponse(w, http.StatusNotFound, s.Errorf("session not found"))
		return
	}

	localAddr := req.URL.Query().Get("local")
	remoteAddr := req.URL.Query().Get("remote")
	remote, err := chshare.DecodeRemote(localAddr + ":" + remoteAddr)
	if err != nil {
		s.jsonErrorResponse(w, http.StatusBadRequest, s.Errorf("invalid request: %s", err))
		return
	}

	// check user permissions
	if session.User != nil && !session.User.HasAccess(remote) {
		s.jsonErrorResponse(w, http.StatusForbidden, s.Errorf("access is not allowed for current session"))
		return
	}

	response := map[string]interface{}{"success": 1}

	// make next steps thread-safe
	session.Lock()
	defer session.Unlock()

	// check if such remote already exists
	if session.HasRemote(remote) {
		response["msg"] = "requested tunnel already exists"
		s.writeJSONResponse(w, http.StatusNoContent, response)
		return
	}

	err = session.StartRemoteTunnel(remote)
	if err != nil {
		s.jsonErrorResponse(w, http.StatusConflict, s.Errorf("can't create tunnel: %s", err))
		return
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}
