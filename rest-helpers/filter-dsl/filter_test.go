package filter_dsl

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ExampleTestSuite struct {
	suite.Suite
}

func filter(rawQuery string, testSubject any) (bool, error) {
	query := strings.Trim(rawQuery, " ")
	if query == "" {
		return true, nil
	}

	kv := strings.Split(rawQuery, ":")

	if len(kv) > 2 {
		return false, fmt.Errorf("query parsing error too many key value separators")
	}
	if len(kv) == 1 {
		return flatFilter(query, testSubject)
	}

	rv := reflect.ValueOf(testSubject)

	if rv.Kind() == reflect.Struct {
		v := rv.FieldByName(kv[0])
		log.Println("bytes", v.Bytes())
		log.Printf("query has key so expected struct, but testSubject is not a struct: %v\n", rv.Kind())
		return false, nil
	}

	if rv.Kind() == reflect.Map {
		log.Printf("query has key so expected struct, but testSubject is not a struct: %v\n", rv.Kind())
		return false, nil
	}

	return false, nil
}

func flatFilter(query string, testSubject any) (bool, error) {
	if query == testSubject {
		return true, nil
	}

	if query == fmt.Sprintf("%v", testSubject) {
		return true, nil
	}
	return false, nil
}

//func (suite *ExampleTestSuite) SetupTest() {
//	suite.VariableThatShouldStartAtFive = 5
//}

func (suite *ExampleTestSuite) TestFilterShouldReturnTrueWhenNoParameters() {
	suite.True(filter("", struct{}{}))
	suite.True(filter(" ", struct{}{}))
}

func (suite *ExampleTestSuite) TestFilterShouldReturnFalseWhenTestSubjectDoesNotMatchQuery() {
	suite.False(filter("5", struct{}{}))
}

func (suite *ExampleTestSuite) TestFilterShouldReturnTrueWhenTestSubjectMatchesQuery() {
	suite.True(filter("5", "5"))
	suite.True(filter(" 5", "5"))
	suite.True(filter("5 ", "5"))
	suite.True(filter("  5", "5"))
	suite.True(filter("5  ", "5"))
	suite.True(filter("  5  ", "5"))
}

func (suite *ExampleTestSuite) TestFilterShouldReturnTrueDisregardingType() {
	suite.True(filter("5", "5"))
	suite.True(filter("5", 5))
}

//// nested structures

func (suite *ExampleTestSuite) TestFilterShouldErrorOnTooManyKeyValueSeparators() {
	_, err := filter("::", "5")
	suite.Error(err)
}

func (suite *ExampleTestSuite) TestFilterShouldFalseWhenTestIsNestedButSubjectIsFlatValue() {
	ok, _ := filter("test:5", "5")
	suite.False(ok)

}

//func (suite *ExampleTestSuite) TestFilterShouldErrorOnEmptyKey() { // or maybe should match any? though a mistake? should match all on * ?
//	_, err := filter(":", "5")
//	suite.Error(err)
//}
//
//func (suite *ExampleTestSuite) TestFilterShouldMatchFields() {
//	suite.True(filter("test-key:5", "5"))
//}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(ExampleTestSuite))
}
