package message

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gregdel/pushover"

	errors2 "github.com/realvnc-labs/rport/server/api/errors"
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

func (s *PushoverService) Send(ctx context.Context, data Data) error {
	body := fmt.Sprintf(`Your RPort 2fa token:<b>%s</b>
requested from %s
<i>with %s</i>
<i>valid for %.0f seconds</i>.`, data.Token, data.RemoteAddress, data.UserAgent, data.TTL.Seconds())
	pMsg := &pushover.Message{
		Title:     data.Title,
		Message:   body,
		HTML:      true,
		Timestamp: time.Now().Unix(),
		Retry:     5 * time.Second,
	}
	pReceiver := pushover.NewRecipient(data.SendTo)
	// TODO: pass ctx when pushover lib will support it
	resp, err := s.p.SendMessage(pMsg, pReceiver)
	if err != nil {
		// ErrHTTPPushover means pushover API call returned 5xx
		if errors.Is(err, pushover.ErrHTTPPushover) {
			return errors2.APIError{
				Message:    "pushover service unavailable",
				HTTPStatus: http.StatusServiceUnavailable,
			}
		}

		if is400(err) {
			return errors2.APIError{
				Err:        err,
				HTTPStatus: http.StatusBadRequest,
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
