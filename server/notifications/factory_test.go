package notifications_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/server/notifications/repository/inmemory"
	"github.com/realvnc-labs/rport/share/refs"
)

type FactoryTestSuite struct {
	suite.Suite
	factory notifications.Factory
	store   *MockStore
}

type MockStore struct {
	notifications []notifications.NotificationDetails
}

func (m *MockStore) Save(ctx context.Context, notification notifications.NotificationDetails) error {
	m.notifications = append(m.notifications, notifications.NotificationDetails{})
	return nil
}

func (m *MockStore) List(ctx context.Context) ([]inmemory.NotificationSummary, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockStore) Details(ctx context.Context, notificationID inmemory.NotificationID) (notifications.NotificationDetails, bool, error) {
	if len(m.notifications) == 0 {
		return notifications.NotificationDetails{}, false, nil
	}
	return m.notifications[0], true, nil
}

func (suite *FactoryTestSuite) SetupTest() {
	suite.store = &MockStore{}
	suite.factory = notifications.NewFactory(suite.store)
}

var alertingIdentifiable = refs.GenerateIdentifiable("Alerting")
var problemIdentifiable = refs.GenerateIdentifiable("Problem")
var expectedOrigin = refs.NewOrigin(alertingIdentifiable, problemIdentifiable)

func (suite *FactoryTestSuite) TestFactoryCreatesNotification() {
	notification := notifications.NotificationData{Target: "smtp", Content: "test-content-mail"}
	ni, err := suite.factory.Dispatch(context.Background(), expectedOrigin, notification)
	suite.NoError(err)
	_, found, err := suite.store.Details(context.Background(), ni)
	suite.True(found)
	// suite.Equal(notifications.NotificationDetails{}, details)
}

func TestFactoryTestSuite(t *testing.T) {
	suite.Run(t, new(FactoryTestSuite))
}
