package message

type Service interface {
	Send(msg, receiver string) error
}
