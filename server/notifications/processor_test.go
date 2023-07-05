package notifications_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/refs"
)

type MockConsumer struct {
	message notifications.NotificationDetails
	waiter  chan struct{}
	fail    atomic.Bool
	target  notifications.Target
}

func (c *MockConsumer) Target() notifications.Target {
	return c.target
}

func (c *MockConsumer) Process(_ context.Context, notification notifications.NotificationDetails) error {
	c.message = notification
	if c.waiter != nil {
		<-c.waiter
		<-c.waiter
	}

	if c.fail.Load() {
		return fmt.Errorf("test-error")
	}

	return nil
}

type ProcessorTestSuite struct {
	suite.Suite
	processor      notifications.Processor
	store          *MockStore
	consumer       *MockConsumer
	consumerScript *MockConsumer
}

func (suite *ProcessorTestSuite) SetupTest() {
	suite.store = NewMockStore()
	suite.consumer = &MockConsumer{target: notifications.TargetMail}
	suite.consumerScript = &MockConsumer{target: notifications.TargetScript}
	suite.processor = notifications.NewProcessor(logger.NewLogger("notifications", logger.NewLogOutput(""), logger.LogLevelInfo), suite.store, suite.consumer, suite.consumerScript)
}

func (suite *ProcessorTestSuite) TestProcessNotificationReceived() {
	queued := suite.SendMail()

	suite.awaitNotificationsProcessed()

	suite.Equal(queued, suite.consumer.message)
}

func (suite *ProcessorTestSuite) awaitNotificationsProcessed() {
	wait := true
	for wait {
		time.Sleep(time.Millisecond * 10)
		ns, err := suite.store.List(context.Background())
		suite.NoError(err)
		for _, n := range ns {
			if n.State == notifications.ProcessingStateQueued || n.State == notifications.ProcessingStateRunning {
				continue
			}
		}
		wait = false
	}

}

func (suite *ProcessorTestSuite) TestProcessNotificationStateDone() {
	queued := suite.SendMail()

	suite.awaitNotificationsProcessed()

	queued.State = notifications.ProcessingStateDone

	out, found, _ := suite.store.Details(context.Background(), queued.ID)
	suite.True(found)
	suite.Equal(queued, out)
}

func (suite *ProcessorTestSuite) TestProcessNotificationStateError() {
	queued := suite.SendMail()

	suite.consumer.fail.Store(true)

	suite.awaitNotificationsProcessed()

	queued.State = notifications.ProcessingStateError
	queued.Out = "test-error"

	out, found, _ := suite.store.Details(context.Background(), queued.ID)
	suite.True(found)
	suite.Equal(queued, out)
}

func (suite *ProcessorTestSuite) TestProcessNotificationDispatch() {
	mail := suite.SendMail()
	script := suite.SendUnknownTarget()

	time.Sleep(time.Millisecond * 10)

	mail.State = notifications.ProcessingStateDone
	out, found, _ := suite.store.Details(context.Background(), mail.ID)
	suite.True(found)
	suite.Equal(mail, out)

	out, found, _ = suite.store.Details(context.Background(), script.ID)
	suite.True(found)
	suite.Equal(script, out)
}

func (suite *ProcessorTestSuite) TestProcessNotificationMultiprocessing() {
	mail := suite.SendMail()
	script := suite.SendScript()

	time.Sleep(time.Millisecond * 10)

	mail.State = notifications.ProcessingStateDone
	out, found, _ := suite.store.Details(context.Background(), mail.ID)
	suite.True(found)
	suite.Equal(mail, out)

	script.State = notifications.ProcessingStateDone
	out, found, _ = suite.store.Details(context.Background(), script.ID)
	suite.True(found)
	suite.Equal(script, out)
}

func (suite *ProcessorTestSuite) TestGracefulShutdown() {
	_ = suite.SendMail()
	suite.NoError(suite.processor.Close())
}

func (suite *ProcessorTestSuite) TestProcessManyNotifications() {
	_ = suite.SendMail()
	second := suite.SendMail()

	suite.awaitNotificationsProcessed()

	second.State = notifications.ProcessingStateDone

	out, found, err := suite.store.Details(context.Background(), second.ID)
	suite.NoError(err)
	suite.True(found)
	suite.Equal(second, out)
}

func (suite *ProcessorTestSuite) SendMail() notifications.NotificationDetails {
	notification := notifications.NotificationData{Target: "smtp", Content: "test-content-mail"}

	queued := notifications.NotificationDetails{
		Origin: problemIdentifiable,
		Target: notifications.TargetMail,
		Data:   notification,
		State:  notifications.ProcessingStateQueued,
		ID:     refs.GenerateIdentifiable(notifications.NotificationType),
	}

	suite.NoError(suite.store.Create(context.Background(), queued))
	return queued
}

func (suite *ProcessorTestSuite) SendScript() notifications.NotificationDetails {
	notification := notifications.NotificationData{Target: "smtp", Content: "test-content-mail"}

	queued := notifications.NotificationDetails{
		Origin: problemIdentifiable,
		Target: notifications.TargetScript,
		Data:   notification,
		State:  notifications.ProcessingStateQueued,
		ID:     refs.GenerateIdentifiable(notifications.NotificationType),
	}

	suite.NoError(suite.store.Create(context.Background(), queued))
	return queued
}

func (suite *ProcessorTestSuite) SendUnknownTarget() notifications.NotificationDetails {
	notification := notifications.NotificationData{Target: "smtp", Content: "test-content-mail"}

	queued := notifications.NotificationDetails{
		Origin: problemIdentifiable,
		Target: "never-pickup",
		Data:   notification,
		State:  notifications.ProcessingStateQueued,
		ID:     refs.GenerateIdentifiable(notifications.NotificationType),
	}

	suite.NoError(suite.store.Create(context.Background(), queued))
	return queued
}
func TestProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}
