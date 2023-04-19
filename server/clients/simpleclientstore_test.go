package clients

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SimpleClientStoreTestSuite struct {
	suite.Suite
}

func (suite *SimpleClientStoreTestSuite) SetupTest() {
	// setup
}

func (suite *SimpleClientStoreTestSuite) TestSimpleClientStore() {
	suite.Equal(1, 5)
}

func TestSimpleClientStoreTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleClientStoreTestSuite))
}
