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

	"github.com/cloudradar-monitoring/rport/db/migration/library"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/server/api/session"
	"github.com/cloudradar-monitoring/rport/server/clients/storedtunnels"
	"github.com/cloudradar-monitoring/rport/server/script"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/random"

	"github.com/gorilla/mux"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/command"
	"github.com/cloudradar-monitoring/rport/server/api/message"
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
	*logger.Logger
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
}

func NewAPIListener(
	server *Server,
	fingerprint string,
) (*APIListener, error) {
	ctx := context.Background()

	config := server.config

	var usersProvider users.Provider
	var err error
	if config.API.AuthFile != "" {
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
		usersProvider, err = users.NewUserDatabase(server.authDB, config.API.AuthUserTable, config.API.AuthGroupTable, config.API.IsTwoFAOn(), logger)
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

	libraryDb, err := sqlite.New(path.Join(config.Server.DataDir, "library.db"), library.AssetNames(), library.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed init library DB instance: %w", err)
	}

	scriptLogger := logger.NewLogger("scripts", config.Logging.LogOutput, config.Logging.LogLevel)
	scriptProvider := script.NewSqliteProvider(libraryDb)
	scriptManager := script.NewManager(scriptProvider, scriptLogger)

	commandProvider := command.NewSqliteProvider(libraryDb)
	commandManager := command.NewManager(commandProvider)

	userService := users.NewAPIService(usersProvider, config.API.IsTwoFAOn())

	a := &APIListener{
		Server:            server,
		Logger:            logger.NewLogger("api-listener", config.Logging.LogOutput, config.Logging.LogLevel),
		fingerprint:       fingerprint,
		httpServer:        chshare.NewHTTPServer(int(config.Server.MaxRequestBytes), chshare.WithTLS(config.API.CertFile, config.API.KeyFile)),
		requestLogOptions: config.InitRequestLogOptions(),
		bannedUsers:       security.NewBanList(time.Duration(config.API.UserLoginWait) * time.Second),
		userService:       userService,
		vaultManager:      vault.NewManager(vaultDBProviderFactory, &vault.Aes256PassManager{}, vaultLogger),
		scriptManager:     scriptManager,
		commandManager:    commandManager,
		storedTunnels:     storedtunnels.New(server.clientDB),
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
			msgSrv = message.NewScriptService(
				config.API.TwoFATokenDelivery,
				config.API.TwoFASendToType,
				config.API.twoFASendToRegexCompiled,
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

	sessionDB, err := session.NewSqliteProvider(path.Join(config.Server.DataDir, "api_sessions.db"))
	if err != nil {
		return nil, err
	}

	a.apiSessions, err = session.NewCache(ctx, sessionDB, defaultTokenLifetime, cleanupAPISessionsInterval)
	if err != nil {
		return nil, err
	}

	a.initRouter()

	return a, nil
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

// lookupUser is used to get the user on every request in auth middleware
func (al *APIListener) lookupUser(r *http.Request) (authorized bool, username string, err error) {
	if basicUser, basicPwd, basicAuthProvided := r.BasicAuth(); basicAuthProvided {
		return al.handleBasicAuth(basicUser, basicPwd)

	}

	if bearerToken, bearerAuthProvided := getBearerToken(r); bearerAuthProvided {
		return al.handleBearerToken(r.Context(), bearerToken)
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

	// skip basic auth with password when 2fa is enabled
	if !al.config.API.IsTwoFAOn() {
		passwordOk := verifyPassword(user.Password, password)
		if passwordOk {
			return true, username, nil
		}
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

func (al *APIListener) handleBearerToken(ctx context.Context, bearerToken string) (bool, string, error) {
	authorized, username, apiSession, err := al.validateBearerToken(ctx, bearerToken)
	if err != nil {
		return false, username, err
	}
	if authorized {
		if err := al.increaseSessionLifetime(ctx, apiSession); err != nil {
			// do not return error since it should respond with 401 instead of 500, just log it
			al.Errorf("Failed to increase jwt token lifetime: %v", err)
		}
	}
	return authorized, username, nil
}

const htpasswdBcryptPrefix = "$2y$"

// validateCredentials returns true if given credentials belong to a user with an access to API.
func (al *APIListener) validateCredentials(username, password string, skipPasswordValidation bool) (bool, error) {
	if username == "" {
		return false, nil
	}

	user, err := al.userService.GetByUsername(username)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %v", err)
	}
	if user == nil && skipPasswordValidation && al.config.API.CreateMissingUsers {
		pswd, err := random.UUID4()
		if err != nil {
			return false, err
		}
		user = &users.User{
			Username: username,
			Password: pswd,
			Groups:   []string{al.config.API.DefaultUserGroup},
		}
		err = al.userService.Change(user, "")
		if err != nil {
			return false, fmt.Errorf("failed to create missing user: %v", err)
		}
	}
	if user == nil {
		return false, nil
	}

	if skipPasswordValidation {
		return true, nil
	}

	return verifyPassword(user.Password, password), nil
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

		authorized, username, err := al.handleBearerToken(r.Context(), token)
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
