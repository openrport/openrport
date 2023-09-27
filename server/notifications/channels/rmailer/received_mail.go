package rmailer

import (
	"strings"

	smtpmock "github.com/mocktools/go-smtp-mock/v2"

	"github.com/openrport/openrport/share/simpleops"
)

type ReceivedMail struct {
	smtpmock.Message
}

func (r ReceivedMail) breakDown() []string {
	return strings.Split(r.MsgRequest(), "\r\n")
}

func (r ReceivedMail) GetTo() []string {
	to, b := simpleops.Find(r.breakDown(), func(s string) bool {
		prefix := strings.HasPrefix(s, "To: ")
		return prefix
	})
	if !b {
		return nil
	}
	rawMails := strings.Split(to, ">, <")
	if len(rawMails) > 0 {
		rawMails[0] = rawMails[0][5:]
		last := len(rawMails) - 1
		rawMails[last] = rawMails[last][:len(rawMails[last])-1]
	}

	return rawMails
}

func (r ReceivedMail) GetContentType() string {
	prefix := "Content-Type: "
	subject, b := simpleops.Find(r.breakDown(), func(s string) bool {
		prefix := strings.HasPrefix(s, prefix)
		return prefix
	})
	if !b {
		return ""
	}

	return strings.TrimPrefix(subject, prefix)
}

func (r ReceivedMail) GetSubject() string {
	prefix := "Subject: "
	subject, b := simpleops.Find(r.breakDown(), func(s string) bool {
		prefix := strings.HasPrefix(s, prefix)
		return prefix
	})
	if !b {
		return ""
	}

	return strings.TrimPrefix(subject, prefix)
}

func (r ReceivedMail) GetContent() string {
	request := r.MsgRequest()
	from := strings.Index(request, "\r\n\r\n")

	return request[from+4 : len(request)-2]
}
