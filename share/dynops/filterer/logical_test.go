package filterer_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/share/dynops/filterer"
)

var newFalse = filterer.NewFalse[bool]()

var newTrue = filterer.NewTrue[bool]()

var newBoolCheck = filterer.NewBool()

type LogicalTestSuite struct {
	suite.Suite
}

func (suite *LogicalTestSuite) SetupTest() {
	var err error
	suite.NoError(err)
}

func (suite *LogicalTestSuite) TestAnd_NoOpsFalse() {
	suite.False(filterer.NewAnd[bool]().Run(true))
}

func (suite *LogicalTestSuite) TestAnd_CheckArgPass() {
	suite.True(filterer.NewAnd[bool](newTrue, newBoolCheck).Run(true))
	suite.False(filterer.NewAnd[bool](newBoolCheck).Run(false))
}

func (suite *LogicalTestSuite) TestAnd_FalseOpsFalse() {
	suite.False(filterer.NewAnd[bool](newFalse).Run(true))
	suite.False(filterer.NewAnd[bool](newFalse, newFalse).Run(true))
	suite.False(filterer.NewAnd[bool](newTrue, newFalse).Run(true))
}

func (suite *LogicalTestSuite) TestAnd_TrueOpsTrue() {
	suite.True(filterer.NewAnd[bool](newTrue).Run(true))
	suite.True(filterer.NewAnd[bool](newTrue, newTrue).Run(true))
}

func (suite *LogicalTestSuite) TestOr_NoOpsFalse() {
	suite.False(filterer.NewOr[bool]().Run(true))
}

func (suite *LogicalTestSuite) TestOr_CheckArgPass() {
	suite.True(filterer.NewOr[bool](newTrue, newBoolCheck).Run(true))
	suite.False(filterer.NewOr[bool](newBoolCheck).Run(false))
}

func (suite *LogicalTestSuite) TestOr_FalseOpsFalse() {
	suite.False(filterer.NewOr[bool](newFalse).Run(true))
	suite.False(filterer.NewOr[bool](newFalse, newFalse).Run(true))

}

func (suite *LogicalTestSuite) TestOr_TrueOpsTrue() {
	suite.True(filterer.NewOr[bool](newTrue).Run(true))
	suite.True(filterer.NewOr[bool](newTrue, newTrue).Run(true))
	suite.True(filterer.NewOr[bool](newFalse, newTrue).Run(true))
}

func TestLogicalTestSuite(t *testing.T) {
	suite.Run(t, new(LogicalTestSuite))
}
