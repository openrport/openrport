package chserver

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/bcrypt"

	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

const (
	DefaultMaxCheckPortTimeout = time.Minute
)

type APIListener struct {
	*chshare.Logger

	fingerprint       string
	connectURL        string
	authFile          string
	jwtSecret         string
	sessionService    *SessionService
	apiSessionRepo    *APISessionRepository
	router            *mux.Router
	httpServer        *chshare.HTTPServer
	docRoot           string
	requestLogOptions *requestlog.Options
	authorizationOn   bool
	userSrv           UserService
	maxRequestBytes   int64
	checkPortTimeout  time.Duration
}

type UserService interface {
	GetByUsername(username string) (*users.User, error)
	Count() (int, error)
}

func NewAPIListener(config *Config, s *SessionService, fingerprint string) (*APIListener, error) {
	var authUsers []*users.User
	var err error
	authorizationOn := false
	// auth-file has precedence over auth
	if config.API.AuthFile != "" {
		authorizationOn = true
		authUsers, err = users.GetUsersFromFile(config.API.AuthFile)
	} else if config.API.Auth != "" {
		authorizationOn = true
		var authUser *users.User
		if authUser, err = parseHTTPAuthStr(config.API.Auth); authUser != nil {
			authUsers = append(authUsers, authUser)
		}
	}
	if err != nil {
		return nil, err
	}

	if config.CheckPortTimeout > DefaultMaxCheckPortTimeout {
		return nil, fmt.Errorf("'check_port_timeout' can not be more than %s", DefaultMaxCheckPortTimeout)
	}

	a := &APIListener{
		Logger:            chshare.NewLogger("api-listener", config.LogOutput, config.LogLevel),
		connectURL:        config.URL,
		fingerprint:       fingerprint,
		authFile:          config.API.AuthFile,
		jwtSecret:         config.API.JWTSecret,
		sessionService:    s,
		apiSessionRepo:    NewAPISessionRepository(),
		httpServer:        chshare.NewHTTPServer(int(config.MaxRequestBytes)),
		docRoot:           config.API.DocRoot,
		requestLogOptions: config.InitRequestLogOptions(),
		userSrv:           users.NewUserRepository(authUsers),
		authorizationOn:   authorizationOn,
		maxRequestBytes:   config.MaxRequestBytes,
		checkPortTimeout:  config.CheckPortTimeout,
	}

	a.initRouter()

	return a, nil
}

func (al *APIListener) Start(addr string) error {
	al.Infof("API Listening on %s...", addr)

	h := http.Handler(http.HandlerFunc(al.handleAPIRequest))
	h = requestlog.WrapWith(h, *al.requestLogOptions)
	err := al.httpServer.GoListenAndServe(addr, h)
	if err != nil {
		return err
	}

	go al.ReloadAPIUsers()

	return nil
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
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<20)
			stackLen := runtime.Stack(buf, false)
			al.Errorf("panic: %v", err)
			al.Errorf("stack: %s", buf[:stackLen])
			al.writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{"error": err})
		}
	}()

	var matchedRoute mux.RouteMatch
	routeExists := al.router.Match(r, &matchedRoute)
	if routeExists {
		r = mux.SetURLVars(r, matchedRoute.Vars) // allows retrieving Vars later from request object
		matchedRoute.Handler.ServeHTTP(w, r)
		return
	}

	if al.docRoot != "" {
		redirectURL := al.docRoot + string(os.PathSeparator) + "index.html"
		middleware.Redirect404(http.FileServer(http.Dir(al.docRoot)), redirectURL).ServeHTTP(w, r)
		return
	}

	w.WriteHeader(404)
	_, _ = w.Write([]byte{})
}

func (al *APIListener) lookupUser(r *http.Request) (authorized bool, username string, err error) {
	if basicUser, basicPwd, basicAuthProvided := r.BasicAuth(); basicAuthProvided {
		authorized, err = al.validateCredentials(basicUser, basicPwd)
		username = basicUser
		return
	}

	if bearerToken, bearerAuthProvided := getBearerToken(r); bearerAuthProvided {
		authorized, username, err = al.handleBearerToken(bearerToken)
	}

	return
}

func (al *APIListener) handleBearerToken(bearerToken string) (bool, string, error) {
	authorized, username, apiSession, err := al.validateBearerToken(bearerToken)
	if err != nil {
		return false, "", err
	}
	if authorized {
		if err := al.increaseSessionLifetime(apiSession); err != nil {
			// do not return error since it should respond with 401 instead of 500, just log it
			al.Errorf("Failed to increase jwt token lifetime: %v", err)
		}
	}
	return authorized, username, nil
}

const htpasswdBcryptPrefix = "$2y$"

// validateCredentials returns true if given credentials belong to a user with an access to API.
func (al *APIListener) validateCredentials(username, password string) (bool, error) {
	if username == "" {
		return false, nil
	}

	user, err := al.userSrv.GetByUsername(username)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %v", err)
	}
	if user == nil {
		return false, nil
	}

	// bcrypt hashed password
	if strings.HasPrefix(user.Password, htpasswdBcryptPrefix) {
		return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil, nil
	}

	// plaintext password, constant time compare is used for security reasons
	return subtle.ConstantTimeCompare([]byte(password), []byte(user.Password)) == 1, nil
}

// parseHTTPAuthStr parses <user>:<password> string, returns (user, nil) or (nil, error)
func parseHTTPAuthStr(basicAuth string) (*users.User, error) {
	if basicAuth == "" {
		return nil, nil
	}

	user, pass := chshare.ParseAuth(basicAuth)
	if user == "" || pass == "" {
		return nil, fmt.Errorf("invalid auth format: expected <user>:<password>, actual %s", basicAuth)
	}

	return &users.User{Username: user, Password: pass}, nil
}

// IsAuthorizationOn returns true if authorization for accessing API is enabled.
func (al *APIListener) IsAuthorizationOn() bool {
	return al.authorizationOn
}
