package dyncopy_test

import (
	"testing"

	"github.com/realvnc-labs/rport/share/dyncopy"
	"github.com/stretchr/testify/suite"
)

type TestStructFrom struct {
	TestNotCopied map[string]int
	TestField     string
	TestFieldInt  int
}

type TestStructTo struct {
	TestField    string
	TestFieldInt int
}

type CopierTestSuite struct {
	suite.Suite
	copier func(TestStructFrom, *TestStructTo)
}

func (suite *CopierTestSuite) SetupTest() {
	var err error
	suite.copier, err = dyncopy.NewCopier[TestStructFrom, TestStructTo](TestStructFrom{}, TestStructTo{}, []dyncopy.FromToPair{})
	suite.NoError(err)
}

func (suite *CopierTestSuite) TestCopier() {
	suite.NotNil(suite.copier)
}

func (suite *CopierTestSuite) TestCopierReturnsErrorIfStructDoesNotHaveAField() {
	_, err := dyncopy.NewCopier[TestStructFrom, TestStructTo](TestStructFrom{}, TestStructTo{}, []dyncopy.FromToPair{dyncopy.NewPair("", "")})
	suite.Errorf(err, "no such field in TestStructFrom: \"\"")

	_, err = dyncopy.NewCopier[TestStructFrom, TestStructTo](TestStructFrom{}, TestStructTo{}, []dyncopy.FromToPair{dyncopy.NewPair("TestField", "NonExistent")})
	suite.Errorf(err, "no such field in TestStructTo: \"NonExistent\"")
}

func (suite *CopierTestSuite) TestCopierErrorsOnTypeMismatch() {
	_, err := dyncopy.NewCopier[TestStructFrom, TestStructTo](TestStructFrom{}, TestStructTo{}, []dyncopy.FromToPair{dyncopy.NewPair("TestField", "TestFieldInt")})
	suite.Errorf(err, "types of fields don't align - copy impossible for fields: TestField, TestFieldInt")
}

func (suite *CopierTestSuite) TestCopierCopiesNothingForEmptyFieldsList() {
	suite.copier(TestStructFrom{}, &TestStructTo{})
}

func (suite *CopierTestSuite) TestCopierCreatesCopierForSinglePair() {
	copier, err := dyncopy.NewCopier[TestStructFrom, TestStructTo](TestStructFrom{}, TestStructTo{}, []dyncopy.FromToPair{dyncopy.NewPair("TestField", "TestField"), dyncopy.NewPair("TestFieldInt", "TestFieldInt")})
	suite.NoError(err)
	from := TestStructFrom{
		TestField:    "test1",
		TestFieldInt: 42,
	}
	to := TestStructTo{}
	copier(from, &to)

	suite.Equal(TestStructTo{
		TestField:    "test1",
		TestFieldInt: 42,
	}, to)
}

func TestCopierTestSuite(t *testing.T) {
	suite.Run(t, new(CopierTestSuite))
}
