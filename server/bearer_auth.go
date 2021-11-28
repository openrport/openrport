package chserver

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/cloudradar-monitoring/rport/server/api/session"
)

const (
	maxTokenLifetime     = 90 * 24 * time.Hour
	defaultTokenLifetime = 10 * time.Minute
	TotPScope            = "totp" // tokens with this scope might be used to pass 2fa or create totp private key for the first time
)

type Token struct {
	Username string `json:"username,omitempty"`
	Targets  string `json:"targets,omitempty"` // comma-sep list of URIs of pages where this token allowed, * means allowed on all pages
	Scope    string `json:"scope,omitempty"`   // validity scope of the token
	jwt.StandardClaims
}

type TokenContext struct {
	AppToken *Token
	RawToken string
	JwtToken *jwt.Token
}

func (al *APIListener) createAuthToken(ctx context.Context, lifetime time.Duration, username, targets, scope string) (string, error) {
	if username == "" {
		return "", errors.New("username cannot be empty")
	}

	claims := Token{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			Id: strconv.FormatUint(rand.Uint64(), 10),
		},
		Targets: targets,
		Scope:   scope,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(al.config.API.JWTSecret))
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(lifetime)
	err = al.apiSessions.Save(ctx, &session.APISession{Token: tokenStr, ExpiresAt: expiresAt})
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func (al *APIListener) increaseSessionLifetime(ctx context.Context, s *session.APISession) error {
	s.ExpiresAt = time.Now().Add(defaultTokenLifetime)
	return al.apiSessions.Save(ctx, s)
}

func (al *APIListener) currentURIMatchesTokenTargets(currentURI, tokenTargetsRaw string) bool {
	currentURI = "/" + strings.Trim(currentURI, "/")
	if tokenTargetsRaw == "" {
		return true
	}

	tokenTargets := strings.Split(tokenTargetsRaw, ",")
	for _, tokenTarget := range tokenTargets {
		if tokenTarget == "*" || currentURI == tokenTarget {
			return true
		}
	}

	return false
}

func (al *APIListener) parseToken(tokenStr string) (tokCtx *TokenContext, err error) {
	appToken := &Token{}
	bearerToken, err := jwt.ParseWithClaims(tokenStr, appToken, func(token *jwt.Token) (i interface{}, err error) {
		return []byte(al.config.API.JWTSecret), nil
	})
	if err != nil {
		// do not return error since it should respond with 401 instead of 500, just log it
		al.Debugf("failed to parse jwt token: %v", err)
		return nil, err
	}

	return &TokenContext{
		AppToken: appToken,
		RawToken: tokenStr,
		JwtToken: bearerToken,
	}, nil
}

func (al *APIListener) validateBearerToken(ctx context.Context, tokCtx *TokenContext, uri string) (bool, *session.APISession, error) {
	if !al.currentURIMatchesTokenTargets(uri, tokCtx.AppToken.Targets) {
		al.Errorf(
			"Token target %s doesn't match the current url %s, so this token is not intended to be used for this page",
			tokCtx.AppToken.Targets,
			uri,
		)
		return false, nil, nil
	}

	if al.bannedUsers.IsBanned(tokCtx.AppToken.Username) {
		return false, nil, ErrTooManyRequests
	}

	if !tokCtx.JwtToken.Valid || tokCtx.AppToken.Username == "" {
		al.Errorf(
			"Token is invalid or user name is empty",
			tokCtx.AppToken.Username,
		)
		return false, nil, nil
	}

	apiSession, err := al.apiSessions.Get(ctx, tokCtx.RawToken)
	if err != nil || apiSession == nil {
		return false, nil, err
	}

	isValidByExpirationTime := apiSession.ExpiresAt.After(time.Now())
	if !isValidByExpirationTime {
		al.Errorf(
			"api session time %v is expired",
			apiSession.ExpiresAt,
		)
		return false, nil, err
	}
	return true, apiSession, nil
}

func getBearerToken(req *http.Request) (string, bool) {
	auth := req.Header.Get("Authorization")
	const prefix = "Bearer "
	// Case insensitive prefix match.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", false
	}
	return auth[len(prefix):], true
}
