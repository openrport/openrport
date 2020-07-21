package chserver

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type APIListener struct {
	*chshare.Logger

	sessionRepo *SessionRepository
	router      *mux.Router
	httpServer  *chshare.HTTPServer
}

func NewAPIListener(config *Config, s *SessionRepository) *APIListener {
	a := &APIListener{
		Logger:      chshare.NewLogger("api-listener"),
		sessionRepo: s,
		httpServer:  chshare.NewHTTPServer(),
	}
	a.Info = true
	a.Debug = config.Verbose

	a.initRouter()

	return a
}

func (al *APIListener) Start(addr string) error {
	al.Infof("API Listening on %s...", addr)

	h := http.Handler(http.HandlerFunc(al.handleAPIRequest))

	if al.Debug {
		o := requestlog.DefaultOptions
		o.TrustProxy = true
		h = requestlog.WrapWith(h, o)
	}
	return al.httpServer.GoListenAndServe(addr, h)
}

func (al *APIListener) Wait() error {
	if al.httpServer == nil {
		return nil
	}
	return al.httpServer.Wait()
}

func (al *APIListener) Close() error {
	if al.httpServer == nil {
		return nil
	}
	return al.httpServer.Close()
}

func (al *APIListener) handleAPIRequest(w http.ResponseWriter, r *http.Request) {
	var matchedRoute mux.RouteMatch
	routeExists := al.router.Match(r, &matchedRoute)
	if routeExists {
		r = mux.SetURLVars(r, matchedRoute.Vars) // allows retrieving Vars later from request object
		matchedRoute.Handler.ServeHTTP(w, r)
		return
	}
	w.WriteHeader(404)
	_, _ = w.Write([]byte{})
}
