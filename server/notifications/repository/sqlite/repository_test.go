package sqlite_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/db/sqlite"
	"github.com/realvnc-labs/rport/server/notifications"
	repo "github.com/realvnc-labs/rport/server/notifications/repository/sqlite"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/refs"
)

var problemIdentifiable = refs.GenerateIdentifiable("Problem")

var testLog = logger.NewLogger("client", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

type RepositoryTestSuite struct {
	suite.Suite
	repository repo.Repository
}

func (suite *RepositoryTestSuite) SetupTest() {
	db, err := sqlite.New(":memory:", repo.AssetNames(), repo.Asset, sqlite.DataSourceOptions{})
	suite.NoError(err)

	suite.repository = repo.NewRepository(db, testLog)
}

func (suite *RepositoryTestSuite) TestRepositoryEmptyListIsEmpty() {
	items, err := suite.repository.List(context.Background(), nil)
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

	notification.State = notifications.ProcessingStateDispatching
	notification.Out = ""

	suite.NoError(suite.repository.SetDispatching(context.Background(), notification.ID.ID()))

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

func (suite *RepositoryTestSuite) TestRepositoryNotificationList() {
	suite.T().Skip()
	suite.CreateNotification()
	notification := suite.CreateNotification()

	notification.State = notifications.ProcessingStateError
	notification.Out = "test-error"

	suite.NoError(suite.repository.SetError(context.Background(), notification.ID.ID(), "test-error"))

	list, err := suite.repository.List(context.Background(), nil)
	suite.NoError(err)
	suite.Equal(list, []notifications.NotificationSummary{})
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
	entities, err := suite.repository.List(context.Background(), nil)
	suite.NoError(err)
	suite.Len(entities, 2)
}

func (suite *RepositoryTestSuite) TestRepositoryRejectNewNotificationsWhenCloseToFullChannel() {
	for i := 0; i < repo.MaxNotificationsQueue; i++ {
		identifiable := refs.GenerateIdentifiable(notifications.NotificationType)
		details := notifications.NotificationDetails{
			RefID: problemIdentifiable,
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
		if err != nil {
			if i > repo.MaxNotificationsQueue*0.5 {
				suite.ErrorContains(err, "rejected")
				return
			}
			suite.NoError(err)
		}
	}
	suite.Fail("should reject when too many notifications wait")
}

func (suite *RepositoryTestSuite) CreateNotification() notifications.NotificationDetails {
	identifiable := refs.GenerateIdentifiable(notifications.NotificationType)
	details := notifications.NotificationDetails{
		RefID: problemIdentifiable,
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
