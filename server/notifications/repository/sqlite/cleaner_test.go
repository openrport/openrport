package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	"github.com/openrport/openrport/db/sqlite"
	db "github.com/openrport/openrport/server/notifications/repository/sqlite"
	"github.com/openrport/openrport/share/logger"
)

type CleanerTestSuite struct {
	suite.Suite
	repository db.Repository
	db         *sqlx.DB
	logger     *logger.Logger
}

func (suite *CleanerTestSuite) SetupTest() {
	var err error
	suite.db, err = sqlite.New(":memory:", db.AssetNames(), db.Asset, sqlite.DataSourceOptions{})
	suite.NoError(err)
	suite.repository = db.NewRepository(suite.db, testLog)
	suite.logger = logger.NewLogger("notifications", logger.NewLogOutput(""), logger.LogLevelInfo)
}

func (suite *CleanerTestSuite) TestCleanerCleansNothingWhenNothing() {
	repository := db.NewRepository(suite.db, testLog)
	c := db.StartCleaner(suite.logger, repository, time.Second, time.Second)
	defer c.Close()

	suite.expectNotifications(0)
}

func (suite *CleanerTestSuite) TestCleanerCleansNothingWhenEverythingIsFresh() {
	suite.createNewNotification()

	c := suite.startCleaner()
	defer c.Close()

	suite.expectNotifications(1)
}

func (suite *CleanerTestSuite) TestCleanerCleansOldNotificationsAfterStart() {
	suite.makeOldNotification()
	suite.createNewNotification()

	c := suite.startCleaner()
	defer c.Close()

	// wait for first initial cleanup to start
	time.Sleep(time.Millisecond * 100)

	suite.expectNotifications(1)

}

func (suite *CleanerTestSuite) TestCleanerCleansOldNotificationsAfterTimeout() {
	suite.createNewNotification()

	c := suite.startCleaner()
	defer c.Close()

	// wait for notification to get old and for cleaner to garbage collect
	time.Sleep(time.Second + time.Millisecond*100)

	suite.expectNotifications(0)

}

func (suite *CleanerTestSuite) createNewNotification() {
	suite.NoError(suite.repository.SetDone(context.Background(), GenerateNotification(), ""))
}

func (suite *CleanerTestSuite) TestCleanerCloses() {
	suite.createNewNotification()

	c := suite.startCleaner()
	_ = c.Close()

	// wait longer then cleaning interval to confirm that message was not deleted hence cleaner must be closed
	time.Sleep(time.Second * 2)

	suite.expectNotifications(1)
}

func TestCleanerTestSuite(t *testing.T) {
	suite.Run(t, new(CleanerTestSuite))
}

func (suite *CleanerTestSuite) startCleaner() db.Closeable {
	repository := db.NewRepository(suite.db, testLog)
	c := db.StartCleaner(suite.logger, repository, time.Second, time.Second)
	return c
}

func (suite *CleanerTestSuite) makeOldNotification() {
	suite.NoError(suite.repository.SetDone(context.Background(), GenerateNotification(), ""))
	time.Sleep(time.Second * 1)
}

func (suite *CleanerTestSuite) expectNotifications(count int) {
	all, err := suite.repository.List(context.Background(), nil)
	suite.NoError(err)
	suite.Len(all, count)
}
