package notifications_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/server/notifications/channels/rmailer"
)

type MockMailer struct {
	body string
}

func (m *MockMailer) Send(to []string, subject string, contentType rmailer.ContentType, body string) error {
	m.body = body
	return nil
}

type MockScriptNotifier struct {
	body string
}

func (m *MockScriptNotifier) Run(script string, recipients []string, subject string, body string) error {
	m.body = body
	return nil
}

type MailTestSuite struct {
	suite.Suite
	notifier           notifications.Notifier
	mockMailer         MockMailer
	mockScriptNotifier MockScriptNotifier
}

func (ts *MailTestSuite) SetupSuite() {
	ts.notifier = notifications.NewNotifier(&ts.mockMailer, &ts.mockScriptNotifier)
}

func (ts *MailTestSuite) TestNotifyMail() {
	_, _ = ts.notifier.Dispatch(notifications.NotificationData{})
}

func (ts *MailTestSuite) TestNotifyDispatchToMail() {
	notification := notifications.NotificationData{Target: "smtp", Content: "test-content-mail"}
	_, err := ts.notifier.Dispatch(notification)
	ts.NoError(err)
	ts.Equal(notification.Content, ts.mockMailer.body)
	ts.NotEqual(notification.Content, ts.mockScriptNotifier.body)
}

func (ts *MailTestSuite) TestMailShouldHaveNiceTemplate() {
	notification := notifications.NotificationData{Target: "smtp", Content: "test-content-mail", ContentType: notifications.ContentTypeTextHTML}
	_, err := ts.notifier.Dispatch(notification)
	ts.NoError(err)
	ts.Contains(ts.mockMailer.body, notification.Content)
	ts.Greater(len(ts.mockMailer.body), len(notification.Content))
}

func (ts *MailTestSuite) TestNotifyDispatchToScript() {
	notification := notifications.NotificationData{Target: "something", Content: "test-content-script"}
	_, err := ts.notifier.Dispatch(notification)
	ts.NoError(err)
	ts.NotEqual(notification.Content, ts.mockMailer.body)
	ts.Equal(notification.Content, ts.mockScriptNotifier.body)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMailTestSuite(t *testing.T) {
	suite.Run(t, new(MailTestSuite))
}
