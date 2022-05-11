package message

import (
	"context"
	"time"
)

type Data struct {
	Title         string
	SendTo        string
	Token         string
	UserAgent     string
	RemoteAddress string
	TTL           time.Duration
}

type Service interface {
	Send(ctx context.Context, data Data) error
	DeliveryMethod() string
	ValidateReceiver(ctx context.Context, receiver string) error
}

type ServiceMock struct {
	ReturnError error
}

func (s *ServiceMock) Send(ctx context.Context, data Data) error {
	return nil
}

func (s *ServiceMock) DeliveryMethod() string {
	return "mock"
}

func (s *ServiceMock) ValidateReceiver(ctx context.Context, receiver string) error {
	return s.ReturnError
}
