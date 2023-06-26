package sqlite_test

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/db/sqlite"
	"github.com/realvnc-labs/rport/server/notifications"
	me "github.com/realvnc-labs/rport/server/notifications/repository/sqlite"
	"github.com/realvnc-labs/rport/share/refs"
)

var alertingIdentifiable = refs.GenerateIdentifiable("Alerting")
var problemIdentifiable = refs.GenerateIdentifiable("Problem")
var expectedOrigin = refs.NewOrigin(alertingIdentifiable, problemIdentifiable)

type RepositoryTestSuite struct {
	suite.Suite
	repository me.Repository
}

func (suite *RepositoryTestSuite) SetupTest() {
	db, err := sqlite.New(":memory:", me.AssetNames(), me.Asset, sqlite.DataSourceOptions{})
	suite.NoError(err)

	suite.repository = me.NewRepository(db)
}

func (suite *RepositoryTestSuite) TestRepositoryEmptyListIsEmpty() {
	items, err := suite.repository.List(context.Background())
	suite.NoError(err)
	suite.Len(items, 0)
}

func (suite *RepositoryTestSuite) TestRepositoryNotificationDoesNotExist() {
	_, found, err := suite.repository.Details(context.Background(), "not-found")
	suite.NoError(err)
	suite.False(found)
}

func (suite *RepositoryTestSuite) TestRepositoryNotificationSavePropagatesToDetails() {
	details := suite.CreateNotification()
	retrieved, found, err := suite.repository.Details(context.Background(), details.ID.ID())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(details, retrieved)
}

func (suite *RepositoryTestSuite) TestRepositoryNotificationRunning() {
	notification := suite.CreateNotification()

	notification.State = notifications.ProcessingStateRunning
	notification.Out = ""

	suite.NoError(suite.repository.SetRunning(context.Background(), notification.ID.ID()))

	retrieved, found, err := suite.repository.Details(context.Background(), notification.ID.ID())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(notification, retrieved)
}

func (suite *RepositoryTestSuite) TestRepositoryNotificationDone() {
	notification := suite.CreateNotification()

	notification.State = notifications.ProcessingStateDone
	notification.Out = ""

	suite.NoError(suite.repository.SetDone(context.Background(), notification.ID.ID()))

	retrieved, found, err := suite.repository.Details(context.Background(), notification.ID.ID())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(notification, retrieved)
}

func (suite *RepositoryTestSuite) TestRepositoryNotificationError() {
	notification := suite.CreateNotification()

	notification.State = notifications.ProcessingStateError
	notification.Out = "test-error"

	suite.NoError(suite.repository.SetError(context.Background(), notification.ID.ID(), "test-error"))

	retrieved, found, err := suite.repository.Details(context.Background(), notification.ID.ID())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(notification, retrieved)
}

func (suite *RepositoryTestSuite) TestRepositoryNotificationStream() {
	notification := suite.CreateNotification()

	stream := suite.repository.NotificationStream(notifications.TargetScript)

	retrieved := <-stream

	suite.Equal(notification, retrieved)
}

func (suite *RepositoryTestSuite) TestRepositoryNotificationListWithEntities() {
	e1 := suite.CreateNotification()
	e2 := suite.CreateNotification()
	log.Println(e1)
	log.Println(e2)
	entities, err := suite.repository.List(context.Background())
	suite.NoError(err)
	suite.Len(entities, 2)
}

func (suite *RepositoryTestSuite) CreateNotification() notifications.NotificationDetails {
	identifiable := refs.GenerateIdentifiable(notifications.NotificationType)
	details := notifications.NotificationDetails{
		Origin: expectedOrigin,
		Data: notifications.NotificationData{
			ContentType: notifications.ContentTypeTextHTML,
			Target:      "test-target",
			Subject:     "test-subject",
			Content:     "test-content",
		},
		State:  notifications.ProcessingStateQueued,
		ID:     identifiable,
		Out:    "test-out",
		Target: "script",
	}
	err := suite.repository.Create(context.Background(), details)
	suite.NoError(err)
	return details
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
