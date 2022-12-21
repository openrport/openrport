package chserver

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"

	"github.com/cloudradar-monitoring/rport/db/migration/api_token"
	"github.com/cloudradar-monitoring/rport/db/migration/library"
	"github.com/cloudradar-monitoring/rport/db/sqlite"

	"github.com/cloudradar-monitoring/rport/server/api/authorization"
	"github.com/cloudradar-monitoring/rport/server/api/session"
	"github.com/cloudradar-monitoring/rport/server/chconfig"
	"github.com/cloudradar-monitoring/rport/server/clients/storedtunnels"
	"github.com/cloudradar-monitoring/rport/server/script"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/command"
	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/bearer"
	"github.com/cloudradar-monitoring/rport/server/vault"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/enums"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/random"
	"github.com/cloudradar-monitoring/rport/share/security"
)

const (
	DefaultMaxCheckPortTimeout = time.Minute
)

type APIListener struct {
	*logger.Logger
	errResponseLogger *logger.Logger

	*Server

	fingerprint       string
	apiSessions       *session.Cache
	router            *mux.Router
	httpServer        *chshare.HTTPServer
	requestLogOptions *requestlog.Options
	accessLogFile     io.WriteCloser
	insecureForTests  bool
	bannedUsers       *security.BanList
	bannedIPs         *security.MaxBadAttemptsBanList
	twoFASrv          TwoFAService

	testDone chan bool // is used only in tests to be able to wait until async task is done

	userService    UserService
	vaultManager   *vault.Manager
	scriptManager  *script.Manager
	tokenManager   *authorization.Manager
	commandManager *command.Manager
	storedTunnels  *storedtunnels.Manager
}

type UserService interface {
	GetAll() ([]*users.User, error)
	GetByUsername(username string) (*users.User, error)
	Change(*users.User, string) error
	Delete(string) error
	ExistGroups([]string) error
	GetProviderType() enums.ProviderSource
	ListGroups() ([]users.Group, error)
	GetGroup(string) (users.Group, error)
	UpdateGroup(string, users.Group) (users.Group, error)
	DeleteGroup(string) error
	CheckPermission(*users.User, string) error
	SupportsGroupPermissions() bool
	GetEffectiveUserPermissions(*users.User) (map[string]bool, error)
}

func NewAPIListener(
	server *Server,
	fingerprint string,
) (*APIListener, error) {
	ctx := context.Background()

	config := server.config

	var usersProvider users.Provider
	var err error

	if isOAuthPermittedUserList(config) {
		if config.API.AuthFile != "" {
			logger := logger.NewLogger("auth-file", config.Logging.LogOutput, config.Logging.LogLevel)
			usersProvider, err = users.NewFileAdapter(logger, users.NewFileManager(config.API.AuthFile))
			if err != nil {
				return nil, err
			}
		} else if config.API.AuthUserTable != "" {
			logger := logger.NewLogger("database", config.Logging.LogOutput, config.Logging.LogLevel)
			usersProvider, err = newAPIAuthDatabase(server, config, logger)
			if err != nil {
				return nil, err
			}
		}
	} else if config.API.AuthFile != "" {
		logger := logger.NewLogger("auth-file", config.Logging.LogOutput, config.Logging.LogLevel)
		usersProvider, err = users.NewFileAdapter(logger, users.NewFileManager(config.API.AuthFile))
		if err != nil {
			return nil, err
		}
	} else if config.API.Auth != "" {
		authUser, e := parseHTTPAuthStr(config.API.Auth)
		if e != nil {
			return nil, e
		}
		// for static user set the admin group
		authUser.Groups = []string{users.Administrators}
		usersProvider = users.NewStaticProvider([]*users.User{authUser})
	} else if config.API.AuthUserTable != "" {
		logger := logger.NewLogger("database", config.Logging.LogOutput, config.Logging.LogLevel)
		usersProvider, err = newAPIAuthDatabase(server, config, logger)
		if err != nil {
			return nil, err
		}
	}

	if config.Server.CheckPortTimeout > DefaultMaxCheckPortTimeout {
		return nil, fmt.Errorf("'check_port_timeout' can not be more than %s", DefaultMaxCheckPortTimeout)
	}

	vaultLogger := logger.NewLogger("vault", config.Logging.LogOutput, config.Logging.LogLevel)

	vaultDBProviderFactory := vault.NewStatefulDbProviderFactory(
		func() (vault.DbProvider, error) {
			return vault.NewSqliteProvider(config, vaultLogger)
		},
		&vault.NotInitDbProvider{},
	)

	// init vault DB if it already exists
	fs := files.NewFileSystem()
	exist, err := fs.Exist(config.GetVaultDBPath())
	if err != nil {
		return nil, fmt.Errorf("failed to check if vault DB %q exists: %v", config.GetVaultDBPath(), err)
	}
	if exist {
		err := vaultDBProviderFactory.Init()
		if err != nil {
			return nil, err
		}
	}

	libraryDb, err := sqlite.New(
		path.Join(config.Server.DataDir, "library.db"),
		library.AssetNames(),
		library.Asset,
		config.Server.GetSQLiteDataSourceOptions(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed init library DB instance: %w", err)
	}

	api_tokenDb, err := sqlite.New(
		path.Join(config.Server.DataDir, "api_token.db"),
		api_token.AssetNames(),
		api_token.Asset,
		config.Server.GetSQLiteDataSourceOptions(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed init api_token DB instance: %w", err)
	}

	scriptLogger := logger.NewLogger("scripts", config.Logging.LogOutput, config.Logging.LogLevel)
	scriptProvider := script.NewSqliteProvider(libraryDb)
	scriptManager := script.NewManager(scriptProvider, scriptLogger)

	commandProvider := command.NewSqliteProvider(libraryDb)
	commandManager := command.NewManager(commandProvider)

	tokenProvider := authorization.NewSqliteProvider(api_tokenDb)
	tokenManager := authorization.NewManager(tokenProvider)

	userService := users.NewAPIService(usersProvider, config.API.IsTwoFAOn(), config.API.PasswordMinLength, config.API.PasswordZxcvbnMinscore)

	a := &APIListener{
		Server:            server,
		Logger:            logger.NewLogger("api-listener", config.Logging.LogOutput, config.Logging.LogLevel),
		fingerprint:       fingerprint,
		httpServer:        chshare.NewHTTPServer(int(config.Server.MaxRequestBytes), chshare.WithTLS(config.API.CertFile, config.API.KeyFile, security.TLSConfig)),
		requestLogOptions: config.InitRequestLogOptions(),
		bannedUsers:       security.NewBanList(time.Duration(config.API.UserLoginWait) * time.Second),
		userService:       userService,
		vaultManager:      vault.NewManager(vaultDBProviderFactory, &vault.Aes256PassManager{}, vaultLogger),
		scriptManager:     scriptManager,
		commandManager:    commandManager,
		tokenManager:      tokenManager,
		storedTunnels:     storedtunnels.New(server.clientDB),
	}

	a.errResponseLogger = server.Logger.Fork("error-response")

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
			msgSrv = message.NewScriptService(
				config.API.TwoFATokenDelivery,
				config.API.TwoFASendToType,
				config.API.TwoFASendToRegexCompiled,
			)
		}

		a.twoFASrv = NewTwoFAService(
			config.API.TwoFATokenTTLSeconds,
			config.API.TwoFASendTimeout,
			userService,
			msgSrv,
		)
		userService.DeliverySrv = msgSrv
		a.Logger.Infof("2FA is enabled via using %s", config.API.TwoFATokenDelivery)
	}

	if config.API.TotPEnabled {
		a.twoFASrv = NewTwoFAService(
			config.API.TwoFATokenTTLSeconds,
			config.API.TwoFASendTimeout,
			userService,
			nil,
		)
		a.Logger.Infof("2FA is enabled via an Authenticator app")
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

	sessionDB, err := session.NewSqliteProvider(path.Join(config.Server.DataDir, "api_sessions.db"), config.Server.GetSQLiteDataSourceOptions())
	if err != nil {
		return nil, err
	}

	a.apiSessions, err = session.NewCache(ctx, bearer.DefaultTokenLifetime, cleanupAPISessionsInterval, sessionDB, nil)
	if err != nil {
		return nil, err
	}

	a.initRouter()

	return a, nil
}

func newAPIAuthDatabase(server *Server, config *chconfig.Config, logger *logger.Logger) (usersProvider *users.UserDatabase, err error) {
	usersProvider, err = users.NewUserDatabase(
		server.authDB,
		config.API.AuthUserTable,
		config.API.AuthGroupTable,
		config.API.AuthGroupDetailsTable,
		config.API.IsTwoFAOn(),
		config.API.TotPEnabled,
		logger,
	)
	return usersProvider, err
}

func (al *APIListener) Start(addr string) error {
	al.Infof("API Listening on %s...", addr)

	err := al.httpServer.GoListenAndServe(addr, al.router)
	if err != nil {
		return err
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
	if al.commandManager != nil {
		g.Go(al.commandManager.Close)
	}

	if al.apiSessions != nil {
		g.Go(al.apiSessions.Close)
	}

	return g.Wait()
}

var ErrTooManyRequests = errors.New("too many requests, please try later")
var ErrThatPasswordHasExpired = errors.New("password has expired, please change your password")

// lookupUser is used to get the user on every request in auth middleware
func (al *APIListener) lookupUser(r *http.Request, isBearerOnly bool) (authorized bool, username string, err error) {
	if !isBearerOnly {
		if basicUser, basicPwd, basicAuthProvided := r.BasicAuth(); basicAuthProvided {
			return al.handleBasicAuth(basicUser, basicPwd)
		}
	}

	if bearerToken, bearerAuthProvided := bearer.GetBearerToken(r); bearerAuthProvided {
		isAuthorized, token, err := al.checkBearerToken(r.Context(), bearerToken, r.URL.Path, r.Method)
		if err != nil {
			return isAuthorized, "", err
		}

		return isAuthorized, token.AppClaims.Username, nil
	}

	// case when no auth method is provided
	if al.bannedUsers.IsBanned("") {
		return false, "", ErrTooManyRequests
	}

	return false, "", nil
}

// handleBasicAuth checks username and password against either user's password or token
func (al *APIListener) handleBasicAuth(username, password string) (authorized bool, name string, err error) {
	if al.bannedUsers.IsBanned(username) {
		return false, username, ErrTooManyRequests
	}

	if username == "" {
		return false, "", nil
	}

	user, err := al.userService.GetByUsername(username)
	if err != nil {
		return false, username, fmt.Errorf("failed to get user: %v", err)
	}
	if user == nil {
		return false, username, nil
	}

	if user.PasswordExpired != nil && *user.PasswordExpired {
		return false, username, ErrThatPasswordHasExpired
	}

	// skip basic auth with password when 2fa is enabled
	if !al.config.API.IsTwoFAOn() && !al.config.API.TotPEnabled {
		passwordOk := verifyPassword(user.Password, password)
		if passwordOk {
			return true, username, nil
		}
	}

	prefix, password, err := authorization.Extract(password)
	if err != nil {
		return false, username, err
	}

	// only check token if we have one saved
	if user.Token != nil && *user.Token != "" {
		tokenOk := verifyPassword(*user.Token, password)
		if tokenOk {
			return true, username, nil
		}
	}

	return false, username, nil
}

func (al *APIListener) checkBearerToken(ctx context.Context, bearerToken, uri, method string) (bool, *bearer.TokenContext, error) {
	tokenCtx, err := bearer.ParseToken(bearerToken, al.config.API.JWTSecret)
	if err != nil {
		al.Debugf("failed to parse jwt token: %v", err)
		return false, nil, err
	}

	if al.bannedUsers.IsBanned(tokenCtx.AppClaims.Username) {
		al.Errorf(
			"User %s is banned",
			tokenCtx.AppClaims.Username,
		)
		return false, nil, ErrTooManyRequests
	}

	authorized, apiSession, err := bearer.ValidateBearerToken(
		ctx,
		tokenCtx,
		uri,
		method,
		al.apiSessions,
		al.Logger)
	if err != nil {
		return false, tokenCtx, err
	}
	if authorized {
		// extend the token lifetime by a short amount so that in-progress activities can complete
		if err := bearer.IncreaseSessionLifetime(ctx, al.apiSessions, apiSession); err != nil {
			// do not return error since it should respond with 401 instead of 500, just log it
			al.Errorf("Failed to increase jwt token lifetime: %v", err)
		}
	}
	return authorized, tokenCtx, nil
}

const htpasswdBcryptPrefix = "$2y$"

// validateCredentials returns true if given credentials belong to a user with access to the API.
func (al *APIListener) validateCredentials(username, password string, skipPasswordValidation bool) (bool, *users.User, error) {
	if username == "" {
		return false, nil, nil
	}

	user, err := al.userService.GetByUsername(username)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get user: %v", err)
	}

	if al.shouldCreateMissingUser(user, skipPasswordValidation) {
		pswd, err := random.UUID4()
		if err != nil {
			return false, nil, err
		}
		user = &users.User{
			Username: username,
			Password: pswd,
			Groups:   []string{al.config.API.DefaultUserGroup},
		}
		err = al.userService.Change(user, "")
		if err != nil {
			return false, user, fmt.Errorf("failed to create missing user: %v", err)
		}
	}

	if user == nil {
		return false, user, nil
	}

	if skipPasswordValidation {
		return true, user, nil
	}

	return verifyPassword(user.Password, password), user, nil
}

func (al *APIListener) shouldCreateMissingUser(user *users.User, skipPasswordValidation bool) bool {
	if user != nil || !skipPasswordValidation {
		return false
	}
	if al.config.API.CreateMissingUsers || !isOAuthPermittedUserList(al.config) {
		return true
	}
	return false
}

func isOAuthPermittedUserList(cfg *chconfig.Config) (is bool) {
	if !cfg.PlusOAuthEnabled() {
		return false
	}
	return cfg.PlusConfig.OAuthConfig.PermittedUserList
}

func verifyPassword(saved, provided string) bool {
	// bcrypt hashed password
	if strings.HasPrefix(saved, htpasswdBcryptPrefix) {
		return bcrypt.CompareHashAndPassword([]byte(saved), []byte(provided)) == nil
	}

	// plaintext password, constant time compare is used for security reasons
	return subtle.ConstantTimeCompare([]byte(saved), []byte(provided)) == 1
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
	errAccessTokenRequired = errors.New("token required")
)

func (al *APIListener) wsAuth(f http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var authorized bool
		var username string
		var err error

		tokenStr := r.URL.Query().Get(WebSocketAccessTokenQueryParam)
		if tokenStr == "" {
			basicUser, basicPwd, basicAuthProvided := r.BasicAuth()

			if basicAuthProvided {
				authorized, username, err = al.handleBasicAuth(basicUser, basicPwd)
			} else {
				if !al.handleBannedIPs(r, false) {
					return
				}
				al.jsonErrorResponse(w, http.StatusUnauthorized, errAccessTokenRequired)
				return
			}
		} else {
			var token *bearer.TokenContext
			authorized, token, err = al.checkBearerToken(r.Context(), tokenStr, r.URL.Path, r.Method)
			if authorized && err == nil {
				username = token.AppClaims.Username
			}
		}

		if err != nil {
			if errors.Is(err, ErrTooManyRequests) {
				al.jsonErrorResponse(w, http.StatusTooManyRequests, err)
				return
			}
			al.jsonErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if !al.handleBannedIPs(r, authorized) {
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
