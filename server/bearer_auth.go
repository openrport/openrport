package chserver

import (
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

func (al *APIListener) createAuthToken(lifetime time.Duration) (string, error) {
	claims := jwt.StandardClaims{
		Id: strconv.FormatUint(rand.Uint64(), 10),
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

func (al *APIListener) validateBearerToken(tokenStr string) (bool, *APISession, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (i interface{}, err error) {
		return []byte(al.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return false, nil, nil
	}

	apiSession, err := al.apiSessionRepo.FindOne(tokenStr)
	if err != nil || apiSession == nil {
		return false, apiSession, err
	}

	return apiSession.ExpiresAt.After(time.Now()), apiSession, nil
}

func getBearerToken(req *http.Request) (string, bool) {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return "", false
	}
	const prefix = "Bearer "
	// Case insensitive prefix match.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", false
	}

	return auth[len(prefix):], true
}
