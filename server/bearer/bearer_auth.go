package bearer

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/realvnc-labs/rport/server/api/session"
	"github.com/realvnc-labs/rport/server/routes"
	"github.com/realvnc-labs/rport/share/logger"
)

const (
	DefaultMaxTokenLifetime = 90 * 24 * time.Hour
	DefaultTokenLifetime    = 10 * time.Minute
)

type AppTokenClaims struct {
	Username  string  `json:"username,omitempty"`
	SessionID int64   `json:"sessionID,omitempty"`
	Scopes    []Scope `json:"scopes,omitempty"`
	jwt.StandardClaims
}

type Scope struct {
	URI     string `json:"uri,omitempty"`
	Method  string `json:"method,omitempty"`
	Exclude bool   `json:"exclude,omitempty"`
}

var ScopesAllExcluding2FaCheck = []Scope{
	{
		URI:    "*",
		Method: "*",
	},
	{
		URI:     routes.AllRoutesPrefix + routes.Verify2FaRoute,
		Method:  "*",
		Exclude: true,
	},
}

var ScopesTotPCreateOnly = []Scope{
	{
		URI:    routes.AllRoutesPrefix + routes.TotPRoutes,
		Method: http.MethodPost,
	},
}

var Scopes2FaCheckOnly = []Scope{
	{
		URI:    routes.AllRoutesPrefix + routes.Verify2FaRoute,
		Method: http.MethodPost,
	},
}

type TokenContext struct {
	AppClaims *AppTokenClaims
	RawToken  string
	JwtToken  *jwt.Token
}

type APISessionUpdater interface {
	Save(ctx context.Context, session session.APISession) (sessionID int64, err error)
}

type APISessionGetter interface {
	Get(ctx context.Context, sessionID int64) (found bool, sessionInfo session.APISession, err error)
}

func CreateAuthToken(
	ctx context.Context,
	sessionUpdater APISessionUpdater,
	JWTSecret string,
	lifetime time.Duration,
	username string,
	scopes []Scope,
	userAgent string,
	remoteAddress string,
) (string, error) {
	if username == "" {
		return "", errors.New("username cannot be empty")
	}

	expiresAt := time.Now().Add(lifetime)

	newSession := session.APISession{
		ExpiresAt:    expiresAt,
		Username:     username,
		LastAccessAt: time.Now(),
		UserAgent:    userAgent,
		IPAddress:    remoteAddress,
	}
	sessionID, err := sessionUpdater.Save(ctx, newSession)
	if err != nil {
		return "", err
	}

	claims := AppTokenClaims{
		Username:  username,
		SessionID: sessionID,
		StandardClaims: jwt.StandardClaims{
			Id: strconv.FormatUint(rand.Uint64(), 10),
		},
		Scopes: scopes,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func IncreaseSessionLifetime(
	ctx context.Context,
	sessionUpdater APISessionUpdater,
	s session.APISession) error {
	newExpirationDate := time.Now().Add(DefaultTokenLifetime)
	if s.ExpiresAt.Before(newExpirationDate) {
		s.ExpiresAt = newExpirationDate
	}
	_, err := sessionUpdater.Save(ctx, s)
	if err != nil {
		return err
	}
	return nil
}

func currentURIMatchesTokenScopes(currentURI, currentMethod string, tokenScopes []Scope) bool {
	// make it compatible with the old tokens which don't have scopes field in jwt
	if len(tokenScopes) == 0 {
		return true
	}
	currentURI = "/" + strings.Trim(currentURI, "/")

	hasAtLeastOneMatch := false
	hasExcludeMatch := false
	for _, tokenScope := range tokenScopes {
		uriMatched := tokenScope.URI == "*" || currentURI == tokenScope.URI
		methodMatched := tokenScope.Method == "*" || currentMethod == tokenScope.Method

		if uriMatched && methodMatched {
			if tokenScope.Exclude {
				hasExcludeMatch = true
			} else {
				hasAtLeastOneMatch = true
			}
		}
	}

	return hasAtLeastOneMatch && !hasExcludeMatch
}

func ParseToken(tokenStr string, JWTSecret string) (tokCtx *TokenContext, err error) {
	appClaims := &AppTokenClaims{}
	bearerToken, err := jwt.ParseWithClaims(tokenStr, appClaims, func(token *jwt.Token) (i interface{}, err error) {
		return []byte(JWTSecret), nil
	})
	if err != nil {
		// do not return error since it should respond with 401 instead of 500, just log it
		return nil, err
	}

	return &TokenContext{
		AppClaims: appClaims,
		RawToken:  tokenStr,
		JwtToken:  bearerToken,
	}, nil
}

func ValidateBearerToken(
	ctx context.Context,
	tokCtx *TokenContext,
	uri, method string,
	apiSessionGetter APISessionGetter,
	l *logger.Logger) (valid bool, sessionInfo session.APISession, err error) {
	if !currentURIMatchesTokenScopes(uri, method, tokCtx.AppClaims.Scopes) {
		l.Errorf(
			"Token scopes %+v don't match with the current url %s[%s], so this token is not intended to be used for this page",
			tokCtx.AppClaims.Scopes,
			method,
			uri,
		)
		return false, session.APISession{}, nil
	}

	if !tokCtx.JwtToken.Valid || tokCtx.AppClaims.Username == "" {
		l.Errorf(
			"Token is invalid or user name is empty",
			tokCtx.AppClaims.Username,
		)
		return false, session.APISession{}, nil
	}

	found, sessionInfo, err := apiSessionGetter.Get(ctx, tokCtx.AppClaims.SessionID)
	if err != nil || !found {
		l.Errorf(
			"Login session not found for %s",
			tokCtx.AppClaims.Username,
		)
		return false, session.APISession{}, err
	}

	isValidByExpirationTime := sessionInfo.ExpiresAt.After(time.Now())
	if !isValidByExpirationTime {
		l.Errorf(
			"api session time %v is expired",
			sessionInfo.ExpiresAt,
		)
		return false, sessionInfo, err
	}
	return true, sessionInfo, nil
}

func GetBearerToken(req *http.Request) (string, bool) {
	auth := req.Header.Get("Authorization")
	const prefix = "Bearer "
	// Case insensitive prefix match.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", false
	}
	return auth[len(prefix):], true
}
