package chserver

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

const (
	DefaultMaxCheckPortTimeout = time.Minute
)

type APIListener struct {
	*chshare.Logger
	*Server

	fingerprint       string
	apiSessionRepo    *APISessionRepository
	router            *mux.Router
	httpServer        *chshare.HTTPServer
	requestLogOptions *requestlog.Options
	userSrv           UserService
	accessLogFile     io.WriteCloser
}

type UserService interface {
	GetByUsername(username string) (*users.User, error)
}

func NewAPIListener(
	server *Server,
	fingerprint string,
) (*APIListener, error) {
	config := server.config
	if config.API.AuthFile != "" && config.API.Auth != "" {
		return nil, errors.New("API: 'auth_file' and 'auth' are both set: expected only one of them ")
	}

	var authUsers []*users.User
	var err error
	if config.API.AuthFile != "" {
		authUsers, err = users.GetUsersFromFile(config.API.AuthFile)
	}
	if config.API.Auth != "" {
		var authUser *users.User
		if authUser, err = parseHTTPAuthStr(config.API.Auth); authUser != nil {
			authUsers = append(authUsers, authUser)
		}
	}
	if err != nil {
		return nil, err
	}

	if config.Server.CheckPortTimeout > DefaultMaxCheckPortTimeout {
		return nil, fmt.Errorf("'check_port_timeout' can not be more than %s", DefaultMaxCheckPortTimeout)
	}

	a := &APIListener{
		Server:            server,
		Logger:            chshare.NewLogger("api-listener", config.Logging.LogOutput, config.Logging.LogLevel),
		fingerprint:       fingerprint,
		apiSessionRepo:    NewAPISessionRepository(),
		httpServer:        chshare.NewHTTPServer(int(config.Server.MaxRequestBytes), chshare.WithTLS(config.API.CertFile, config.API.KeyFile)),
		requestLogOptions: config.InitRequestLogOptions(),
		userSrv:           users.NewUserCache(authUsers),
	}

	if config.API.AccessLogFile != "" {
		accessLogFile, err := os.OpenFile(config.API.AccessLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		a.accessLogFile = accessLogFile
	}

	a.initRouter()

	return a, nil
}

func (al *APIListener) Start(addr string) error {
	al.Infof("API Listening on %s...", addr)

	h := http.Handler(http.HandlerFunc(al.handleAPIRequest))
	h = requestlog.WrapWith(h, *al.requestLogOptions)
	if al.accessLogFile != nil {
		h = handlers.CombinedLoggingHandler(al.accessLogFile, h)
	}
	err := al.httpServer.GoListenAndServe(addr, h)
	if err != nil {
		return err
	}

	// Only reload api users from file if file auth is set
	if al.config.API.AuthFile != "" {
		go al.ReloadAPIUsers()
	}

	return nil
}

func (al *APIListener) Wait() error {
	if al.httpServer == nil {
		return nil
	}
	return al.httpServer.Wait()
}

func (al *APIListener) Close() error {
	g := &errgroup.Group{}
	if al.httpServer != nil {
		g.Go(al.httpServer.Close)
	}
	if al.accessLogFile != nil {
		g.Go(al.accessLogFile.Close)
	}
	return g.Wait()
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

	docRoot := al.config.API.DocRoot
	if docRoot != "" {
		middleware.Rewrite404(http.FileServer(http.Dir(docRoot)), "/").ServeHTTP(w, r)
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
	if al.config == nil {
		return false
	}
	return al.config.API.AuthFile != "" || al.config.API.Auth != ""
}
