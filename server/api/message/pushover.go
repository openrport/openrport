package message

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gregdel/pushover"

	"github.com/cloudradar-monitoring/rport/server/api/errors"
)

type PushoverService struct {
	p *pushover.Pushover
}

func NewPushoverService(apiToken string) *PushoverService {
	return &PushoverService{
		p: pushover.New(apiToken),
	}
}

const pushoverAPISuccessStatus = 1

func (s *PushoverService) Send(msg, receiver string) error {
	pMsg := pushover.NewMessageWithTitle(msg, "Rport 2FA token")
	pReceiver := pushover.NewRecipient(receiver)
	resp, err := s.p.SendMessage(pMsg, pReceiver)
	if err != nil {
		// pushover custom errors from github.com/gregdel/pushover can be identified by 'pushover' string in it
		isPushoverCustomErr := strings.Contains(err.Error(), "pushover")
		err = fmt.Errorf("failed to send 2fa token: %v", err)
		if isPushoverCustomErr {
			return errors.APIError{
				Err:  err,
				Code: http.StatusBadRequest,
			}
		}
		return err
	}

	if resp.Status == pushoverAPISuccessStatus {
		return nil
	}

	return errors.APIError{
		Message: fmt.Sprintf("failed to send 2fa token, request: %s, status: %v, receipt: %s, errors: %v", resp.ID, resp.Status, resp.Receipt, resp.Errors),
		Code:    http.StatusBadRequest,
	}
}
