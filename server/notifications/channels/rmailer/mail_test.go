package rmailer_test

import (
	"fmt"
	"testing"

	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/server/notifications/channels/rmailer"
)

type MailTestSuite struct {
	suite.Suite
	server *smtpmock.Server
	mailer rmailer.Mailer
}

func (ts *MailTestSuite) SetupSuite() {
	ts.server = smtpmock.New(smtpmock.ConfigurationAttr{
		//LogToStdout:              true, // for debugging (especially connection)
		//LogServerActivity:        true, // for debugging (especially connection)
		MultipleMessageReceiving: true,
		//		PortNumber:        33334, // randomly generated
	})
	ts.NoError(ts.server.Start())

	ts.mailer = rmailer.NewRMailer(rmailer.Config{
		Host:     "localhost",
		Port:     ts.server.PortNumber(),
		Domain:   "example.com",
		From:     "test@example.com",
		TLS:      false,
		AuthType: rmailer.AuthTypeNone,
		NoNoop:   true,
	})

	if err := ts.server.Start(); err != nil {
		fmt.Println(err)
	}
}

func (ts *MailTestSuite) TestMailSent() {

	ts.NoError(ts.mailer.Send([]string{"tina.recipient@example.com", "just+fff@some.mail.com"}, "test subject!", rmailer.ContentTypeTextHTML, "test\r\n\r\n<b>content</b>"))

	ts.ExpectMessage([]string{"tina.recipient@example.com", "just+fff@some.mail.com"}, "test subject!", "text/html; charset=UTF-8", "test\r\n\r\n<b>content</b>")

}

func (ts *MailTestSuite) TestMailSMTPConfigCompatibility() {

	config, err := rmailer.ConfigFromSMTPConfig(chconfig.SMTPConfig{
		Server:       "testsmtp.somedomain.com",
		AuthUsername: "test-user",
		AuthPassword: "test-password",
		SenderEmail:  "test@somedomain.com",
		Secure:       true,
	})

	ts.NoError(err)

	ts.Equal(rmailer.Config{
		Host:     "testsmtp.somedomain.com",
		Port:     -1,
		Domain:   "somedomain.com",
		From:     "test@somedomain.com",
		TLS:      true,
		AuthType: rmailer.AuthTypeUserPass,
		AuthUserPass: rmailer.AuthUserPass{
			User: "test-user",
			Pass: "test-password",
		},
		NoNoop: false,
	}, config)
}

func (ts *MailTestSuite) ExpectedMessages(count int) bool {
	return ts.Len(ts.server.Messages(), count)
}

func (ts *MailTestSuite) ExpectMessage(to []string, subject string, contentType string, content string) {
	if !ts.ExpectedMessages(1) {
		return
	}
	receivedMail := rmailer.ReceivedMail{Message: ts.server.Messages()[0]}

	ts.Equal(to, receivedMail.GetTo())

	ts.Equal(subject, receivedMail.GetSubject())

	ts.Equal(contentType, receivedMail.GetContentType())

	ts.Equal(content, receivedMail.GetContent())

}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMailTestSuite(t *testing.T) {
	suite.Run(t, new(MailTestSuite))
}
