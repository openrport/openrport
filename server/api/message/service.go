package message

import "context"

type Service interface {
	Send(ctx context.Context, title, msg, receiver string) error
	DeliveryMethod() string
	ValidateReceiver(ctx context.Context, receiver string) error
}

type ServiceMock struct {
	ReturnError error
}

func (s *ServiceMock) Send(ctx context.Context, title, msg, receiver string) error {
	return nil
}

func (s *ServiceMock) DeliveryMethod() string {
	return "mock"
}

func (s *ServiceMock) ValidateReceiver(ctx context.Context, receiver string) error {
	return s.ReturnError
}
