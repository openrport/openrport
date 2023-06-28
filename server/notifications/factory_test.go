package notifications_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/share/refs"
)

type DispatcherTestSuite struct {
	suite.Suite
	dispatcher notifications.Dispatcher
	store      *MockStore
}

func (suite *DispatcherTestSuite) SetupTest() {
	suite.store = NewMockStore()
	suite.dispatcher = notifications.NewDispatcher(suite.store)
}

var alertingIdentifiable = refs.GenerateIdentifiable("Alerting")
var problemIdentifiable = refs.GenerateIdentifiable("Problem")
var expectedOrigin = refs.NewOrigin(alertingIdentifiable, problemIdentifiable)

func (suite *DispatcherTestSuite) TestDispatcherCreatesNotification() {
	notification := notifications.NotificationData{Target: "smtp", Content: "test-content-mail"}
	ni, err := suite.dispatcher.Dispatch(context.Background(), expectedOrigin, notification)
	suite.NoError(err)
	details, found, err := suite.store.Details(context.Background(), ni)
	suite.NoError(err)
	suite.True(found)
	suite.Equal(notifications.NotificationDetails{
		Origin: expectedOrigin,
		Data:   notification,
		State:  notifications.ProcessingStateQueued,
		ID:     ni,
		Target: notifications.TargetMail,
	}, details)
}

func TestDispatcherTestSuite(t *testing.T) {
	suite.Run(t, new(DispatcherTestSuite))
}
