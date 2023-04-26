package message

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	email2 "github.com/realvnc-labs/rport/share/email"
)

type URLService struct {
	senderURL string
	baseURL   string
}

func NewURLService(senderURL, baseURL string) *URLService {
	return &URLService{
		senderURL: senderURL,
		baseURL:   baseURL,
	}
}

func (s *URLService) Send(ctx context.Context, data Data) error {
	values := url.Values{}
	values.Set("email", data.SendTo)
	values.Set("token", data.Token)
	values.Set("ttl", strconv.Itoa(int(data.TTL.Seconds())))
	values.Set("user_agent", data.UserAgent)
	values.Set("remote_address", data.RemoteAddress)
	values.Set("url", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.senderURL, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return errors.New(resp.Status)
	}
	return nil
}

func (s *URLService) DeliveryMethod() string {
	return "url"
}

func (s *URLService) ValidateReceiver(ctx context.Context, email string) error {
	return email2.Validate(email)
}
