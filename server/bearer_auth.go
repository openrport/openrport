package chserver

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/cloudradar-monitoring/rport/server/api/session"
)

const (
	maxTokenLifetime     = 90 * 24 * time.Hour
	defaultTokenLifetime = 10 * time.Minute
)

type Token struct {
	Username string  `json:"username,omitempty"`
	Scopes   []Scope `json:"scopes,omitempty"`
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
		URI:     allRoutesPrefix + verify2FaRoute,
		Method:  "*",
		Exclude: true,
	},
}

var ScopesTotPCreateOnly = []Scope{
	{
		URI:    allRoutesPrefix + totPRoutes,
		Method: http.MethodPost,
	},
}

var Scopes2FaCheckOnly = []Scope{
	{
		URI:    allRoutesPrefix + verify2FaRoute,
		Method: http.MethodPost,
	},
}

type TokenContext struct {
	AppToken *Token
	RawToken string
	JwtToken *jwt.Token
}

func (al *APIListener) createAuthToken(
	ctx context.Context,
	lifetime time.Duration,
	username string,
	scopes []Scope,
) (string, error) {
	if username == "" {
		return "", errors.New("username cannot be empty")
	}

	claims := Token{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			Id: strconv.FormatUint(rand.Uint64(), 10),
		},
		Scopes: scopes,
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
	newExpirationDate := time.Now().Add(defaultTokenLifetime)
	if s.ExpiresAt.Before(newExpirationDate) {
		s.ExpiresAt = newExpirationDate
	}
	return al.apiSessions.Save(ctx, s)
}

func (al *APIListener) currentURIMatchesTokenScopes(currentURI, currentMethod string, tokenScopes []Scope) bool {
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

func (al *APIListener) validateBearerToken(ctx context.Context, tokCtx *TokenContext, uri, method string) (bool, *session.APISession, error) {
	if !al.currentURIMatchesTokenScopes(uri, method, tokCtx.AppToken.Scopes) {
		al.Errorf(
			"Token scopes %+v don't match with the current url %s[%s], so this token is not intended to be used for this page",
			tokCtx.AppToken.Scopes,
			method,
			uri,
		)
		return false, nil, nil
	}

	if al.bannedUsers.IsBanned(tokCtx.AppToken.Username) {
		al.Errorf(
			"User %s is banned",
			tokCtx.AppToken.Username,
		)
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
		al.Errorf(
			"Login session not found for %s",
			tokCtx.AppToken.Username,
		)
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
