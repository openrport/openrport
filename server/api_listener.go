package chserver

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type APIListener struct {
	*chshare.Logger

	authUser       string
	authPassword   string
	jwtSecret      string
	sessionRepo    *SessionRepository
	apiSessionRepo *APISessionRepository
	router         *mux.Router
	httpServer     *chshare.HTTPServer
	docRoot        string
}

func NewAPIListener(config *Config, s *SessionRepository) (*APIListener, error) {
	authUser, authPassword, err := parseHTTPAuthStr(config.APIAuth)
	if err != nil {
		return nil, err
	}

	a := &APIListener{
		Logger:         chshare.NewLogger("api-listener"),
		authUser:       authUser,
		authPassword:   authPassword,
		jwtSecret:      config.APIJWTSecret,
		sessionRepo:    s,
		apiSessionRepo: NewAPISessionRepository(),
		httpServer:     chshare.NewHTTPServer(),
		docRoot:        config.DocRoot,
	}
	a.Info = true
	a.Debug = config.Verbose

	a.initRouter()

	return a, nil
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

	if al.docRoot != "" {
		http.FileServer(http.Dir(al.docRoot)).ServeHTTP(w, r)
		return
	}

	w.WriteHeader(404)
	_, _ = w.Write([]byte{})
}

func (al *APIListener) handleAuthorization(w http.ResponseWriter, r *http.Request) bool {
	if al.authUser == "" || al.authPassword == "" {
		return true
	}

	basicUser, basicPwd, basicAuthProvided := r.BasicAuth()
	bearerToken, bearerAuthProvided := getBearerToken(r)

	var err error
	var authorized bool
	if basicAuthProvided {
		authorized = al.validateUserPasswordPair(basicUser, basicPwd)
	} else if bearerAuthProvided {
		authorized, err = al.handleBearerToken(bearerToken)
		if err != nil {
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return false
		}
	}

	if !authorized {
		w.Header().Set("WWW-Authenticate", `Basic realm="rportd-api"`)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte{})
	}

	return authorized
}

func (al *APIListener) handleBearerToken(bearerToken string) (bool, error) {
	authorized, apiSession, err := al.validateBearerToken(bearerToken)
	if err != nil {
		return false, err
	}
	if authorized {
		err = al.increaseSessionLifetime(apiSession)
		if err != nil {
			return true, err
		}
	}
	return authorized, nil
}

func (al *APIListener) validateUserPasswordPair(basicUser, basicPwd string) bool {
	return subtle.ConstantTimeCompare([]byte(basicUser), []byte(al.authUser)) == 1 &&
		subtle.ConstantTimeCompare([]byte(basicPwd), []byte(al.authPassword)) == 1
}

// parseHTTPAuthStr parses <user>:<password> string, returns (user, password, nil) or an error
func parseHTTPAuthStr(basicAuth string) (string, string, error) {
	if basicAuth == "" {
		return "", "", nil
	}

	parts := strings.Split(basicAuth, ":")
	if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
		return "", "", fmt.Errorf("can't parse basic-auth string")
	}

	return parts[0], parts[1], nil
}
