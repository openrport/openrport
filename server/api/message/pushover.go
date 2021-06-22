package message

import (
	"context"
	"errors"
	"fmt"
	"net/http"

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

func (s *PushoverService) Send(ctx context.Context, title, msg, receiver string) error {
	pMsg := pushover.NewMessageWithTitle(msg, title)
	pReceiver := pushover.NewRecipient(receiver)
	// TODO: pass ctx when pushover lib will support it
	resp, err := s.p.SendMessage(pMsg, pReceiver)
	if err != nil {
		// ErrHTTPPushover means pushover API call returned 5xx
		if errors.Is(err, pushover.ErrHTTPPushover) {
			return errors2.APIError{
				Message: "pushover service unavailable",
				Code:    http.StatusServiceUnavailable,
			}
		}

		if is400(err) {
			return errors2.APIError{
				Err:  err,
				Code: http.StatusBadRequest,
			}
		}

		return err
	}

	if resp != nil && resp.Status != pushoverAPISuccessStatus {
		return fmt.Errorf("failed to send msg, pushover response: %+v", *resp)
	}

	return nil
}

func (s *PushoverService) DeliveryMethod() string {
	return "pushover"
}

func (s *PushoverService) ValidateReceiver(ctx context.Context, pushoverUserKey string) error {
	if pushoverUserKey == "" {
		return errors.New("pushover user key cannot be empty")
	}

	r := pushover.NewRecipient(pushoverUserKey)
	// TODO: pass ctx when pushover lib will support it
	resp, err := s.p.GetRecipientDetails(r)
	if err != nil {
		return fmt.Errorf("failed to validate pushover user key: %w", err)
	}

	if resp != nil && resp.Status != pushoverAPISuccessStatus {
		return fmt.Errorf("failed to validate user key, pushover response: %+v", *resp)
	}

	return nil
}

var pushover400Errs = []error{
	pushover.ErrEmptyToken,
	pushover.ErrEmptyURL,
	pushover.ErrEmptyRecipientToken,
	pushover.ErrInvalidRecipientToken,
	pushover.ErrInvalidRecipient,
	pushover.ErrInvalidPriority,
	pushover.ErrInvalidToken,
	pushover.ErrMessageEmpty,
	pushover.ErrMessageTitleTooLong,
	pushover.ErrMessageTooLong,
	pushover.ErrMessageAttachmentTooLarge,
	pushover.ErrMessageURLTitleTooLong,
	pushover.ErrMessageURLTooLong,
	pushover.ErrMissingAttachment,
	pushover.ErrMissingEmergencyParameter,
	pushover.ErrInvalidDeviceName,
	pushover.ErrEmptyReceipt,
}

func is400(err error) bool {
	if errors.As(err, &pushover.Errors{}) {
		return true
	}

	for _, curErr := range pushover400Errs {
		if errors.Is(err, curErr) {
			return true
		}
	}

	return false
}
