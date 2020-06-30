package chserver

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func NewAPIRouter(s *Server) *mux.Router {
	r := mux.NewRouter()
	sub := r.PathPrefix("/api/v1").Subrouter()
	sub.HandleFunc("/status", s.handleGetStatus).Methods(http.MethodGet)
	sub.HandleFunc("/sessions", s.handleGetSessions).Methods(http.MethodGet)
	sub.HandleFunc("/sessions/{ID}/tunnels", s.handlePutSessionTunnel).Methods(http.MethodPut)
	return r
}

func (s *Server) writeJSONResponse(w http.ResponseWriter, response interface{}) {
	b, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, string(b)); err != nil {
		s.Infof("error writing response: %s", err)
	}
}

func (s *Server) handleGetStatus(w http.ResponseWriter, req *http.Request) {
	s.writeJSONResponse(w, map[string]interface{}{
		"version":        chshare.BuildVersion,
		"sessions_count": len(s.sessions),
	})
}

func (s *Server) handleGetSessions(w http.ResponseWriter, req *http.Request) {
	var result = make([]*ClientSession, 0)
	for _, c := range s.sessions {
		result = append(result, c)
	}
	s.writeJSONResponse(w, result)
}

func (s *Server) handlePutSessionTunnel(w http.ResponseWriter, req *http.Request) {
	// TODO: implement next
	// vars := mux.Vars(req)
	// vars["ID"]
}
