package message

type Service interface {
	Send(title, msg, receiver string) error
	DeliveryMethod() string
}
