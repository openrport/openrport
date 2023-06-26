package notifications_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/db/sqlite"
	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/server/notifications/channels/rmailer"
	me "github.com/realvnc-labs/rport/server/notifications/repository/sqlite"
)

type NotificationsIntegrationTestSuite struct {
	suite.Suite
	dispatcher   notifications.Dispatcher
	store        me.Repository
	server       *smtpmock.Server
	runner       notifications.Processor
	mailConsumer notifications.Consumer
}

func (suite *NotificationsIntegrationTestSuite) SetupTest() {
	db, err := sqlite.New(":memory:", me.AssetNames(), me.Asset, sqlite.DataSourceOptions{})
	suite.NoError(err)
	suite.store = me.NewRepository(db)
	suite.dispatcher = notifications.NewDispatcher(suite.store)
	suite.server = smtpmock.New(smtpmock.ConfigurationAttr{
		//LogToStdout:              true, // for debugging (especially connection)
		//LogServerActivity:        true, // for debugging (especially connection)
		MultipleMessageReceiving: true,
		// PortNumber:               33334, // randomly generated
	})

	if err := suite.server.Start(); err != nil {
		fmt.Println(err)
	}

	suite.mailConsumer = rmailer.NewConsumer(rmailer.NewRMailer(rmailer.Config{
		Host:     "localhost",
		Port:     suite.server.PortNumber(),
		Domain:   "example.com",
		From:     "test@example.com",
		TLS:      false,
		AuthType: rmailer.AuthTypeNone,
		NoNoop:   true,
	}))

	suite.runner = notifications.NewProcessor(suite.store, suite.mailConsumer)

}

func (suite *NotificationsIntegrationTestSuite) TestDispatcherCreatesNotification() {
	notification := notifications.NotificationData{
		Target:      "smtp",
		Recipients:  []string{"stefan.tester@example.com"},
		Subject:     "test-subject",
		Content:     "test-content-mail",
		ContentType: notifications.ContentTypeTextHTML,
	}
	nid, err := suite.dispatcher.Dispatch(context.Background(), expectedOrigin, notification)
	suite.NoError(err)
	time.Sleep(time.Millisecond * 100)

	d, found, err := suite.store.Details(context.Background(), nid.ID())
	suite.T().Log(d, found, err)
	suite.ExpectedMessages(1)
}

func (suite *NotificationsIntegrationTestSuite) ExpectedMessages(count int) bool {
	return suite.Len(suite.server.Messages(), count)
}

func (suite *NotificationsIntegrationTestSuite) ExpectMessage(to []string, subject string, contentType string, content string) {
	if !suite.ExpectedMessages(1) {
		return
	}
	receivedMail := rmailer.ReceivedMail{Message: suite.server.Messages()[0]}

	suite.Equal(to, receivedMail.GetTo())

	suite.Equal(subject, receivedMail.GetSubject())

	suite.Equal(contentType, receivedMail.GetContentType())

	suite.Equal(content, receivedMail.GetContent())

}

func TestNotificationsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationsIntegrationTestSuite))
}
