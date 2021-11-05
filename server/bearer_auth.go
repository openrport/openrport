package chserver

import (
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	maxTokenLifetime     = 90 * 24 * time.Hour
	defaultTokenLifetime = 10 * time.Minute
)

type Token struct {
	Username string `json:"username,omitempty"`
	jwt.StandardClaims
}

func (al *APIListener) createAuthToken(lifetime time.Duration, username, subject string) (string, error) {
	if username == "" {
		return "", errors.New("username cannot be empty")
	}

	claims := Token{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			Id: strconv.FormatUint(rand.Uint64(), 10),
			Subject: subject,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(al.config.API.JWTSecret))
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(lifetime)
	err = al.apiSessionRepo.Save(&APISession{tokenStr, expiresAt})
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func (al *APIListener) increaseSessionLifetime(s *APISession) error {
	newExpirationDate := s.ExpiresAt.Add(defaultTokenLifetime)
	if time.Now().After(s.ExpiresAt) {
		newExpirationDate = time.Now().Add(defaultTokenLifetime)
	}
	s.ExpiresAt = newExpirationDate
	return al.apiSessionRepo.Save(s)
}

func (al *APIListener) subjectFromRequest(r *http.Request) string {
	return "/" + strings.Trim(r.URL.Path, "/")
}

func (al *APIListener) validateBearerToken(tokenStr, curSubject string) (bool, string, *APISession, error) {
	tk := &Token{}
	token, err := jwt.ParseWithClaims(tokenStr, tk, func(token *jwt.Token) (i interface{}, err error) {
		return []byte(al.config.API.JWTSecret), nil
	})
	if err != nil {
		// do not return error since it should respond with 401 instead of 500, just log it
		al.Debugf("failed to parse jwt token: %v", err)
		return false, "", nil, nil
	}

	if tk.StandardClaims.Subject != "*" &&  tk.StandardClaims.Subject != curSubject {
		al.Errorf(
			"Token subject %s doesn't match the current url %s, so this token is not intended to be used for this page",
			tk.StandardClaims.Subject,
			curSubject,
		)
		return false, tk.Username, nil, nil
	}

	if al.bannedUsers.IsBanned(tk.Username) {
		al.Errorf(
			"User %s is banned",
			tk.Username,
		)
		return false, tk.Username, nil, ErrTooManyRequests
	}

	if !token.Valid || tk.Username == "" {
		al.Errorf(
			"Token is invalid or user name is empty",
			tk.Username,
		)
		return false, "", nil, nil
	}

	apiSession, err := al.apiSessionRepo.FindOne(tokenStr)
	if err != nil || apiSession == nil {
		return false, "", nil, err
	}

	isValidByExpirationTime := apiSession.ExpiresAt.After(time.Now())
	if !isValidByExpirationTime {
		al.Errorf(
			"api session time %v is expired",
			apiSession.ExpiresAt,
		)
		return false, "", nil, err
	}
	return true, tk.Username, apiSession, nil
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
