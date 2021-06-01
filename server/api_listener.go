package chserver

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cloudradar-monitoring/rport/server/script"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/vault"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/enums"
	"github.com/cloudradar-monitoring/rport/share/security"
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
	insecureForTests  bool
	bannedUsers       *security.BanList
	bannedIPs         *security.MaxBadAttemptsBanList
	twoFASrv          TwoFAService

	testDone      chan bool // is used only in tests to be able to wait until async task is done
	usersService  *users.APIService
	vaultManager  *vault.Manager
	scriptManager *script.Manager
}

type UserService interface {
	GetByUsername(username string) (*users.User, error)
}

func NewAPIListener(
	server *Server,
	fingerprint string,
) (*APIListener, error) {
	config := server.config

	var userService UserService
	var usersProviderType enums.ProviderSource
	var userDB *users.UserDatabase
	var err error
	usersFromFileProvider := &users.FileManager{
		FileAccessLock: sync.Mutex{},
	}
	if config.API.AuthFile != "" {
		usersFromFileProvider.FileName = config.API.AuthFile
		authUsers, e := usersFromFileProvider.ReadUsersFromFile()
		if e != nil {
			return nil, e
		}
		userService = users.NewUserCache(authUsers)
		usersProviderType = enums.ProviderSourceFile
	} else if config.API.Auth != "" {
		authUser, e := parseHTTPAuthStr(config.API.Auth)
		if e != nil {
			return nil, e
		}
		userService = users.NewUserCache([]*users.User{authUser})
		usersProviderType = enums.ProviderSourceStatic
	} else if config.API.AuthUserTable != "" {
		logger := chshare.NewLogger("database", config.Logging.LogOutput, config.Logging.LogLevel)
		userDB, err = users.NewUserDatabase(server.db, config.API.AuthUserTable, config.API.AuthGroupTable, config.API.IsTwoFAOn(), logger)
		if err != nil {
			return nil, err
		}
		userService = userDB
		usersProviderType = enums.ProviderSourceDB
	}

	if config.Server.CheckPortTimeout > DefaultMaxCheckPortTimeout {
		return nil, fmt.Errorf("'check_port_timeout' can not be more than %s", DefaultMaxCheckPortTimeout)
	}

	vaultLogger := chshare.NewLogger("vault", config.Logging.LogOutput, config.Logging.LogLevel)

	vaultDBProviderFactory := vault.NewStatefulDbProviderFactory(
		func() (vault.DbProvider, error) {
			return vault.NewSqliteProvider(config.Vault, vaultLogger)
		},
		&vault.NotInitDbProvider{},
	)

	scriptLogger := chshare.NewLogger("scripts", config.Logging.LogOutput, config.Logging.LogLevel)
	scriptDb, err := script.NewSqliteProvider(path.Join(config.Server.DataDir, "scripts.db"), scriptLogger)
	if err != nil {
		return nil, err
	}

	scriptManager := script.NewManager(scriptDb, scriptLogger)

	a := &APIListener{
		Server:            server,
		Logger:            chshare.NewLogger("api-listener", config.Logging.LogOutput, config.Logging.LogLevel),
		fingerprint:       fingerprint,
		apiSessionRepo:    NewAPISessionRepository(),
		httpServer:        chshare.NewHTTPServer(int(config.Server.MaxRequestBytes), chshare.WithTLS(config.API.CertFile, config.API.KeyFile)),
		requestLogOptions: config.InitRequestLogOptions(),
		userSrv:           userService,
		bannedUsers:       security.NewBanList(time.Duration(config.API.UserLoginWait) * time.Second),
		usersService: &users.APIService{
			ProviderType: usersProviderType,
			FileProvider: usersFromFileProvider,
			DB:           userDB,
			TwoFAOn:      config.API.IsTwoFAOn(),
		},
		vaultManager:  vault.NewManager(vaultDBProviderFactory, &vault.Aes256PassManager{}, vaultLogger),
		scriptManager: scriptManager,
	}

	if config.API.IsTwoFAOn() {
		var msgSrv message.Service
		switch config.API.TwoFATokenDelivery {
		case "pushover":
			msgSrv = message.NewPushoverService(config.Pushover.APIToken)
		case "smtp":
			msgSrv, err = message.NewSMTPService(
				config.SMTP.Server,
				config.SMTP.AuthUsername,
				config.SMTP.AuthPassword,
				config.SMTP.SenderEmail,
				config.SMTP.Secure,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to init smtp service: %v", err)
			}
		default:
			return nil, fmt.Errorf("unknown 2fa delivery: %s", config.API.TwoFATokenDelivery)
		}

		a.twoFASrv = NewTwoFAService(config.API.TwoFATokenTTLSeconds, userService, msgSrv)
		a.usersService.DeliverySrv = msgSrv
		a.Logger.Infof("2FA is enabled via using %s", config.API.TwoFATokenDelivery)
	}

	if config.API.MaxFailedLogin > 0 && config.API.BanTime > 0 {
		a.bannedIPs = security.NewMaxBadAttemptsBanList(
			config.API.MaxFailedLogin,
			time.Duration(config.API.BanTime)*time.Second,
			a.Logger,
		)
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

	if al.vaultManager != nil {
		g.Go(al.vaultManager.Close)
	}

	if al.scriptManager != nil {
		g.Go(al.scriptManager.Close)
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

var ErrTooManyRequests = errors.New("too many requests, please try later")

func (al *APIListener) lookupUser(r *http.Request) (authorized bool, username string, err error) {
	// skip basic auth when 2fa is enabled
	if !al.config.API.IsTwoFAOn() {
		if basicUser, basicPwd, basicAuthProvided := r.BasicAuth(); basicAuthProvided {
			if al.bannedUsers.IsBanned(basicUser) {
				return false, basicUser, ErrTooManyRequests
			}
			authorized, err = al.validateCredentials(basicUser, basicPwd)
			username = basicUser
			return
		}
	}

	if bearerToken, bearerAuthProvided := getBearerToken(r); bearerAuthProvided {
		authorized, username, err = al.handleBearerToken(bearerToken)
		return
	}

	// case when no auth method is provided
	if al.bannedUsers.IsBanned("") {
		return false, "", ErrTooManyRequests
	}

	return
}

func (al *APIListener) handleBearerToken(bearerToken string) (bool, string, error) {
	authorized, username, apiSession, err := al.validateBearerToken(bearerToken)
	if err != nil {
		return false, username, err
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

const WebSocketAccessTokenQueryParam = "access_token"

var (
	errUnauthorized        = errors.New("unauthorized")
	errAccessTokenRequired = errors.New("access token required")
)

func (al *APIListener) wsAuth(f http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get(WebSocketAccessTokenQueryParam)
		if token == "" {
			if !al.handleBannedIPs(w, r, false) {
				return
			}
			al.jsonErrorResponse(w, http.StatusUnauthorized, errAccessTokenRequired)
			return
		}

		authorized, username, err := al.handleBearerToken(token)
		if err != nil {
			if errors.Is(err, ErrTooManyRequests) {
				al.jsonErrorResponse(w, http.StatusTooManyRequests, err)
				return
			}
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if !al.handleBannedIPs(w, r, authorized) {
			return
		}

		if !authorized || username == "" {
			al.bannedUsers.Add(username)
			al.jsonErrorResponse(w, http.StatusUnauthorized, errUnauthorized)
			return
		}

		newCtx := api.WithUser(r.Context(), username)
		f.ServeHTTP(w, r.WithContext(newCtx))
	}
}
