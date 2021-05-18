package message

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gregdel/pushover"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

// PushoverService is a service that uses Pushover API to send messages via Pushover.net.
type PushoverService struct {
	p *pushover.Pushover
}

func NewPushoverService(apiToken string) *PushoverService {
	return &PushoverService{
		p: pushover.New(apiToken),
	}
}

const pushoverAPISuccessStatus = 1

func (s *PushoverService) Send(title, msg, receiver string) error {
	pMsg := pushover.NewMessageWithTitle(msg, title)
	pReceiver := pushover.NewRecipient(receiver)
	resp, err := s.p.SendMessage(pMsg, pReceiver)
	if err != nil {
		// pushover custom errors from github.com/gregdel/pushover can be identified by 'pushover' string in it
		isPushoverCustomErr := strings.Contains(err.Error(), "pushover")
		if isPushoverCustomErr {
			return errors2.APIError{
				Err:  err,
				Code: http.StatusBadRequest,
			}
		}
		return err
	}

	if resp.Status == pushoverAPISuccessStatus {
		return nil
	}

	return errors2.APIError{
		Message: fmt.Sprintf("failed to send msg, request: %s, status: %v, receipt: %s, errors: %v", resp.ID, resp.Status, resp.Receipt, resp.Errors),
		Code:    http.StatusBadRequest,
	}
}

func (s *PushoverService) DeliveryMethod() string {
	return "pushover"
}

func (s *PushoverService) ValidateReceiver(pushoverUserKey string) error {
	if pushoverUserKey == "" {
		return errors.New("pushover user key cannot be empty")
	}

	r := pushover.NewRecipient(pushoverUserKey)
	_, err := s.p.GetRecipientDetails(r)
	if err != nil {
		return fmt.Errorf("failed to validate pushover user key: %w", err)
	}

	return nil
}
