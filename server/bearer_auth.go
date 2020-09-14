package chserver

import (
	"errors"
	"fmt"
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

func (al *APIListener) createAuthToken(lifetime time.Duration, username string) (string, error) {
	if username == "" {
		return "", errors.New("username cannot be empty")
	}

	claims := Token{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			Id: strconv.FormatUint(rand.Uint64(), 10),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(al.jwtSecret))
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

func (al *APIListener) validateBearerToken(tokenStr string) (bool, string, *APISession, error) {
	tk := &Token{}
	token, err := jwt.ParseWithClaims(tokenStr, tk, func(token *jwt.Token) (i interface{}, err error) {
		return []byte(al.jwtSecret), nil
	})
	if err != nil {
		return false, "", nil, fmt.Errorf("failed to parse jwt token: %v", err)
	}
	if !token.Valid || tk.Username == "" {
		return false, "", nil, nil
	}

	apiSession, err := al.apiSessionRepo.FindOne(tokenStr)
	if err != nil || apiSession == nil {
		return false, "", nil, err
	}

	return apiSession.ExpiresAt.After(time.Now()), tk.Username, apiSession, nil
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
