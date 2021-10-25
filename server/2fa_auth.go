package chserver

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"sync"
	"time"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/share/security"
)

type TwoFAService struct {
	TokenTTL    time.Duration
	MsgSrv      message.Service
	UserSrv     UserService
	SendTimeout time.Duration

	tokensByUser map[string]*expirableToken
	mu           sync.RWMutex
}

func NewTwoFAService(tokenTTLSeconds int, sendTimeout time.Duration, userSrv UserService, msgSrv message.Service) TwoFAService {
	return TwoFAService{
		TokenTTL:     time.Duration(tokenTTLSeconds) * time.Second,
		UserSrv:      userSrv,
		MsgSrv:       msgSrv,
		SendTimeout:  sendTimeout,
		tokensByUser: make(map[string]*expirableToken),
	}
}

type expirableToken struct {
	token  string
	expiry time.Time
}

const twoFATokenLength = 6

// TODO: add tests
func (srv *TwoFAService) SendToken(ctx context.Context, username string) (sendTo string, err error) {
	ctx, cancel := context.WithTimeout(ctx, srv.SendTimeout)
	defer cancel()

	if username == "" {
		return "", errors2.APIError{
			Message:    "username cannot be empty",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	user, err := srv.UserSrv.GetByUsername(username)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors2.APIError{
			Message:    fmt.Sprintf("user with username %s not found", username),
			HTTPStatus: http.StatusNotFound,
		}
	}

	if user.TwoFASendTo == "" {
		return "", errors2.APIError{
			Message:    "no two_fa_send_to set for this user",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	token, err := security.NewRandomToken(twoFATokenLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate 2fa token: %wv", err)
	}

	data := message.Data{
		SendTo:  user.TwoFASendTo,
		Token:   token,
		TTL:     srv.TokenTTL,
		Title:   "Rport 2FA",
		Message: fmt.Sprintf("Verification code: %s (valid %s)", token, srv.TokenTTL),
	}
	if err := srv.MsgSrv.Send(ctx, data); err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return "", fmt.Errorf("failed to send 2fa verification code: %w", err)
	}

	srv.mu.Lock()
	srv.tokensByUser[username] = &expirableToken{
		token:  token,
		expiry: time.Now().Add(srv.TokenTTL),
	}
	srv.mu.Unlock()

	return user.TwoFASendTo, nil
}

func (srv *TwoFAService) SetTotPLoginSession(username string, loginSessionTTL time.Duration) {
	srv.mu.Lock()
	srv.tokensByUser[username] = &expirableToken{
		expiry: time.Now().Add(loginSessionTTL),
	}
	srv.mu.Unlock()
}

func (srv *TwoFAService) ValidateTotPCode(username, code, secretKey string) error {
	srv.mu.RLock()
	t := srv.tokensByUser[username]
	defer srv.mu.RUnlock()

	if t == nil {
		return errors2.APIError{
			Message:    "login request not found for provided username",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	if time.Now().After(t.expiry) {
		return errors2.APIError{
			Message:    "login request expired",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	if !CheckTotPCode(code, secretKey) {
		return errors2.APIError{
			Message:    "invalid token",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	return nil
}

// TODO: add tests
func (srv *TwoFAService) ValidateToken(username, token string) error {
	srv.mu.RLock()
	t := srv.tokensByUser[username]
	defer srv.mu.RUnlock()

	if t == nil {
		return errors2.APIError{
			Message:    "2fa token not found for provided username",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	if time.Now().After(t.expiry) {
		return errors2.APIError{
			Message:    "2fa token expired",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	if subtle.ConstantTimeCompare([]byte(t.token), []byte(token)) != 1 {
		return errors2.APIError{
			Message:    "invalid token",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	return nil
}
