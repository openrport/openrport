package filterer

import (
	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/share/query"
)

type FiltererTestSuite struct {
	suite.Suite
}

func (suite *FiltererTestSuite) SetupTest() {
	var err error
	suite.NoError(err)
}

func (suite *FiltererTestSuite) TestFilterer_nilOptionsShouldAlwaysBeTrue() {
	//comparator := NewFieldCoparator()
	//suite.True(comparator.Run(false))
}

//func TestFiltererTestSuite(t *testing.T) {
//	suite.Run(t, new(FiltererTestSuite))
//}

func (suite *FiltererTestSuite) genOptions(filters []query.FilterOption) *query.ListOptions {
	options := &query.ListOptions{
		Sorts:      nil,
		Filters:    filters,
		Fields:     nil,
		Pagination: nil,
	}
	return options
}
