package simpleops_test

import (
	"github.com/realvnc-labs/rport/share/simpleops"
	"github.com/stretchr/testify/suite"
	"testing"
)

type FilterTestSuite struct {
	suite.Suite
}

func (f *FilterTestSuite) TestEmptySlice() {
	org := []string{}

	f.Equal([]string(nil), simpleops.FilterSlice(org, func(s string) bool {
		return false
	}))
}

func (f *FilterTestSuite) TestFilterOut() {
	org := []string{"test", "test", "test"}

	f.Equal([]string(nil), simpleops.FilterSlice(org, func(s string) bool {
		return false
	}))
}

func (f *FilterTestSuite) TestFilterIn() {
	org := []string{"test", "test", "test"}

	f.Equal(org, simpleops.FilterSlice(org, func(s string) bool {
		return true
	}))
}

func TestFilterTestSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}
