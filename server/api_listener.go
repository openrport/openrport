package chserver

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"

	"github.com/realvnc-labs/rport/db/migration/api_token"
	"github.com/realvnc-labs/rport/db/migration/library"
	"github.com/realvnc-labs/rport/db/sqlite"
	rportplus "github.com/realvnc-labs/rport/plus"
	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/server/notifications/channels/rmailer"
	"github.com/realvnc-labs/rport/server/notifications/channels/scriptRunner"
	me "github.com/realvnc-labs/rport/server/notifications/repository/sqlite"

	"github.com/realvnc-labs/rport/server/api/authorization"
	"github.com/realvnc-labs/rport/server/api/session"
	"github.com/realvnc-labs/rport/server/clients/storedtunnels"
	"github.com/realvnc-labs/rport/server/script"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/command"
	"github.com/realvnc-labs/rport/server/api/message"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/bearer"
	"github.com/realvnc-labs/rport/server/vault"

	extperm "github.com/realvnc-labs/rport/plus/capabilities/extendedpermission"
	chshare "github.com/realvnc-labs/rport/share"
	"github.com/realvnc-labs/rport/share/enums"
	"github.com/realvnc-labs/rport/share/files"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/random"
	"github.com/realvnc-labs/rport/share/security"
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

	mu                     sync.RWMutex
	notificationsStorage   me.Repository
	notificationsProcessor notifications.Processor
	notificationsDB        *sqlx.DB
}

func (al *APIListener) Log() (l *logger.Logger) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	return al.Logger
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
	GetEffectiveUserExtendedPermissions(*users.User) ([]extperm.PermissionParams, []extperm.PermissionParams)
}

func NewAPIListener(
	server *Server,
	fingerprint string,
) (*APIListener, error) {
	ctx := context.Background()

	config := server.config

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

	db, err := sqlite.New(
		path.Join(config.Server.DataDir, "notifications.db"),
		me.AssetNames(),
		me.Asset,
		config.Server.GetSQLiteDataSourceOptions(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bootstrap api: %v", err)
	}

	store := me.NewRepository(db)
	// dispatcher := notifications.NewDispatcher(store)
	scriptConsumer := scriptRunner.NewConsumer()

	notificationConsumers := []notifications.Consumer{scriptConsumer}
	smtpConfig, err := rmailer.ConfigFromSMTPConfig(config.SMTP)
	if err != nil {
		server.Logger.Errorf("failed to bootstrap smtp notifications: %v", err)
		mailConsumer := rmailer.NewConsumer(rmailer.NewRMailer(smtpConfig))
		notificationConsumers = append(notificationConsumers, mailConsumer)
	}

	runner := notifications.NewProcessor(logger.NewLogger("notifications", config.Logging.LogOutput, config.Logging.LogLevel), store, notificationConsumers...)

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

	apiTokenDb, err := sqlite.New(
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

	tokenProvider := authorization.NewSqliteProvider(apiTokenDb)
	tokenManager := authorization.NewManager(tokenProvider)

	userService, err := users.NewAPIServiceFromConfig(server.authDB, config)
	if err != nil {
		return nil, fmt.Errorf("failed init api users service: %w", err)
	}

	var HTTPServerOptions []chshare.ServerOption
	if config.API.CertFile != "" && config.API.KeyFile != "" {
		HTTPServerOptions = []chshare.ServerOption{chshare.WithTLS(config.API.CertFile, config.API.KeyFile, security.TLSConfig(config.API.TLSMin))}
	}
	// no need for TLS on the api listener when using caddy for API access
	if config.CaddyEnabled() && config.Caddy.APIReverseProxyEnabled() {
		HTTPServerOptions = nil
	}

	if config.API.EnableAcme {
		server.acme.AddHost(config.API.BaseURL)
		tlsConfig := server.acme.ApplyTLSConfig(security.TLSConfig(config.API.TLSMin))
		HTTPServerOptions = []chshare.ServerOption{
			chshare.WithTLS("", "", tlsConfig),
		}
	}

	allog := logger.NewLogger("api-listener", config.Logging.LogOutput, config.Logging.LogLevel)
	a := &APIListener{
		Server:                 server,
		Logger:                 allog,
		fingerprint:            fingerprint,
		httpServer:             chshare.NewHTTPServer(int(config.API.MaxRequestBytes), allog, HTTPServerOptions...),
		requestLogOptions:      config.InitRequestLogOptions(),
		bannedUsers:            security.NewBanList(time.Duration(config.API.UserLoginWait) * time.Second),
		userService:            userService,
		vaultManager:           vault.NewManager(vaultDBProviderFactory, &vault.Aes256PassManager{}, vaultLogger),
		scriptManager:          scriptManager,
		commandManager:         commandManager,
		tokenManager:           tokenManager,
		storedTunnels:          storedtunnels.New(server.clientDB),
		notificationsStorage:   store,
		notificationsProcessor: runner,
		notificationsDB:        db,
	}

	a.errResponseLogger = allog.Fork("error-response")

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
			if _, err := exec.LookPath(config.API.TwoFATokenDelivery); err == nil {
				msgSrv = message.NewScriptService(
					config.API.TwoFATokenDelivery,
					config.API.TwoFASendToType,
					config.API.TwoFASendToRegexCompiled,
				)
			} else {
				msgSrv = message.NewURLService(config.API.TwoFATokenDelivery, config.API.BaseURL)
			}
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

func (al *APIListener) Start(ctx context.Context, addr string) error {
	al.Infof("API Listening on %s...", addr)

	err := al.httpServer.GoListenAndServe(ctx, addr, al.router)
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

	g.Go(al.notificationsStorage.Close)
	g.Go(al.notificationsProcessor.Close)
	g.Go(al.notificationsDB.Close)

	return g.Wait()
}

var ErrTooManyRequests = errors.New("too many requests, please try later")
var ErrThatPasswordHasExpired = errors.New("password has expired, please change your password")
var ErrCantLoadThatToken = errors.New("there was a problem accessing that token with the provided prefix")
var ErrPrefixNotFound = errors.New("there is no token with that prefix")
var ErrInvalidScopeOfThatToken = errors.New("the scope of the provided token is not authorized for this operation")
var ErrThatTokenHasExpired = errors.New("the provided token has expired")

// lookupUser is used to get the user on every request in auth middleware
func (al *APIListener) lookupUser(r *http.Request, isBearerOnly bool) (authorized bool, username string, err error) {
	if !isBearerOnly {
		if basicUser, basicPwd, basicAuthProvided := r.BasicAuth(); basicAuthProvided {
			return al.handleBasicAuth(r.Context(), r.Method, r.URL.Path, basicUser, basicPwd)
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
func (al *APIListener) handleBasicAuth(ctx context.Context, httpverb, urlpath, username, password string) (authorized bool, name string, err error) {
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

	// only check token if we have one saved == I can't know if I have the token for this operation until I check the prefix inside the db
	//   only check token if the prefix gives me a match with the db
	// TODO: this type of tokens "User tokens", meant to be used by scripts - used in place of the password at each request - should be renamed "passwords" or "long lived passwords" or "encrypted long lived passwords"
	prefix, password, err := authorization.Extract(password)
	if err != nil {
		return false, username, nil
	}
	userToken, err := al.tokenManager.Get(ctx, username, prefix)
	if err != nil {
		return false, username, err
	}

	if userToken != nil {
		if userToken.ExpiresAt != nil {
			if userToken.ExpiresAt.Before(time.Now()) {
				return false, username, nil
			}
		}
		tokenOk := verifyPassword(userToken.Token, password)
		if tokenOk {
			switch userToken.Scope {
			case authorization.APITokenRead:
				if httpverb == "GET" && !strings.Contains(urlpath, "/ws") {
					return true, username, nil
				}
			case authorization.APITokenReadWrite:
				return true, username, nil
			case authorization.APITokenClientsAuth:
				if strings.Contains(urlpath, "clients-auth") {
					return true, username, nil
				}
			}
			return false, username, ErrInvalidScopeOfThatToken
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
	if al.config.API.CreateMissingUsers || !rportplus.IsOAuthPermittedUserList(al.config.PlusConfig) {
		return true
	}
	return false
}

func verifyPassword(saved, provided string) bool {
	// bcrypt hashed password
	if strings.HasPrefix(saved, htpasswdBcryptPrefix) {
		return bcrypt.CompareHashAndPassword([]byte(saved), []byte(provided)) == nil
	}

	// plaintext password, constant time compare is used for security reasons
	return subtle.ConstantTimeCompare([]byte(saved), []byte(provided)) == 1
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
				authorized, username, err = al.handleBasicAuth(r.Context(), r.Method, r.URL.Path, basicUser, basicPwd)
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
