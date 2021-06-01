package message

type Service interface {
	Send(title, msg, receiver string) error
	DeliveryMethod() string
	ValidateReceiver(receiver string) error
}

type ServiceMock struct {
	ReturnError error
}

func (s *ServiceMock) Send(title, msg, receiver string) error {
	return nil
}

func (s *ServiceMock) DeliveryMethod() string {
	return "mock"
}

func (s *ServiceMock) ValidateReceiver(receiver string) error {
	return s.ReturnError
}
