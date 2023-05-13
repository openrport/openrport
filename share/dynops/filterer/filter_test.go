package filterer_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/share/dynops/filterer"
	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/random"
)

type FiltererTestSuite struct {
	suite.Suite
}

func (suite *FiltererTestSuite) SetupTest() {
	var err error
	suite.NoError(err)
}

func (suite *FiltererTestSuite) TestFilterer_nilOptionsShouldAlwaysBeTrue() {
	filter, err := filterer.CompileFromQueryListOptions[bool](nil)
	suite.NoError(err)
	suite.True(filter.Run(false))
}

func (suite *FiltererTestSuite) TestFilterer_nilFilterShouldBeTrue() {

	options := genOptions(nil)

	filter, err := filterer.CompileFromQueryListOptions[bool](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(false))
}

func (suite *FiltererTestSuite) TestFilterer_emptyFilterShouldBeTrue() {

	options := genOptions([]query.FilterOption{})

	filter, err := filterer.CompileFromQueryListOptions[bool](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(false))
}

type TestStruct struct {
	SomeField string
	SomeInt   int
	SomeSlice []string
	SomeMap   map[string]string
}

func (suite *FiltererTestSuite) TestFilterer_simpleEquation() {

	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeField"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{"some-value"},
		ValuesLogicalOperator: "",
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeField: "some-value"}))
	suite.False(filter.Run(TestStruct{SomeField: "wrong-value"}))
}

func (suite *FiltererTestSuite) TestFilterer_testInt() {

	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeInt"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{"5"},
		ValuesLogicalOperator: "",
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeInt: 5}))
	suite.False(filter.Run(TestStruct{SomeInt: 3}))
}

func (suite *FiltererTestSuite) TestFilterer_testGt() {

	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeInt"},
		Operator:              query.FilterOperatorTypeGT,
		Values:                []string{"15"},
		ValuesLogicalOperator: "",
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeInt: 3}))
	suite.False(filter.Run(TestStruct{SomeInt: 50}))
}

func (suite *FiltererTestSuite) TestFilterer_testMultiple() {

	options := genOptions([]query.FilterOption{
		{
			Column:                []string{"SomeInt"},
			Operator:              query.FilterOperatorTypeEQ,
			Values:                []string{"5"},
			ValuesLogicalOperator: "",
		},
		{
			Column:                []string{"SomeField"},
			Operator:              query.FilterOperatorTypeEQ,
			Values:                []string{"some-value"},
			ValuesLogicalOperator: "",
		},
	})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeInt: 5, SomeField: "some-value"}))
	suite.False(filter.Run(TestStruct{SomeInt: 5, SomeField: "other-value"}))
	suite.False(filter.Run(TestStruct{SomeInt: 3, SomeField: "some-value"}))
}

func (suite *FiltererTestSuite) TestFilterer_errorOnNonExistentField() {

	options := genOptions([]query.FilterOption{
		{
			Column:                []string{"non-existent"},
			Operator:              query.FilterOperatorTypeEQ,
			Values:                []string{"5"},
			ValuesLogicalOperator: "",
		},
	})

	_, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.ErrorContains(err, "field: non-existent does not exist on type: filterer_test.TestStruct on filter: 0")
}

func (suite *FiltererTestSuite) TestFilterer_errorOnNonExistentOperator() {

	options := genOptions([]query.FilterOption{
		{
			Column:                []string{"SomeField"},
			Operator:              "non-existent",
			Values:                []string{"5"},
			ValuesLogicalOperator: "",
		},
	})

	_, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.ErrorContains(err, "invalid string operator: non-existent on filter: 0")
}

func (suite *FiltererTestSuite) TestFilterer_sliceSimple() {

	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeSlice"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{"some-stuff"},
		ValuesLogicalOperator: "",
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeSlice: []string{"wrong-value", "some-stuff"}}))
	suite.True(filter.Run(TestStruct{SomeSlice: []string{"some-stuff"}}))
	suite.False(filter.Run(TestStruct{SomeSlice: []string{"wrong-value", "other-value"}}))
	suite.False(filter.Run(TestStruct{SomeSlice: []string{"wrong-value"}}))
}

func (suite *FiltererTestSuite) TestFilterer_mapSimple() {

	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeMap"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{"key: val"},
		ValuesLogicalOperator: "",
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeMap: map[string]string{"otherKey": "otherVAl", "key": "val"}}))
	suite.True(filter.Run(TestStruct{SomeMap: map[string]string{"key": "val"}}))
	suite.False(filter.Run(TestStruct{SomeMap: map[string]string{"key": "bad-val"}}))
}

func (suite *FiltererTestSuite) TestFilterer_multipleValuesOr() {

	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeField"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{"failing", "matching"},
		ValuesLogicalOperator: query.FilterLogicalOperatorTypeOR,
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeField: "matching"}))
	suite.False(filter.Run(TestStruct{SomeField: "not-matching"}))
}

func (suite *FiltererTestSuite) TestFilterer_multipleValuesAnd() {

	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeSlice"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{"matching1", "matching2"},
		ValuesLogicalOperator: query.FilterLogicalOperatorTypeAND,
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeSlice: []string{"matching2", "matching1"}}))
	suite.False(filter.Run(TestStruct{SomeSlice: []string{"matching1"}}))
}

func (suite *FiltererTestSuite) TestFilterer_stringSubMatching() {

	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeField"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{"match*"},
		ValuesLogicalOperator: query.FilterLogicalOperatorTypeAND,
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeField: "matching"}))
	suite.False(filter.Run(TestStruct{SomeField: "failing"}))
}

func (suite *FiltererTestSuite) TestStringMatcher_empty() {
	txt := ""
	parts := []string{}
	suite.False(filterer.StringMather(txt, parts))
}

func (suite *FiltererTestSuite) TestStringMatcher_emptyParts() {
	txt := "some-txt"
	parts := []string{}
	suite.False(filterer.StringMather(txt, parts))
}

func (suite *FiltererTestSuite) TestStringMatcher_notMatching() {
	txt := "some-txt"
	parts := []string{"something-else"}
	suite.False(filterer.StringMather(txt, parts))
}

func (suite *FiltererTestSuite) TestStringMatcher_partExistsButNotFromTheBeginning() {
	txt := "some-txt"
	parts := []string{"txt"}
	suite.False(filterer.StringMather(txt, parts))
}

func (suite *FiltererTestSuite) TestStringMatcher_simpleMatching() {
	txt := "some-txt"
	parts := []string{"some-txt"}
	suite.True(filterer.StringMather(txt, parts))
}

func (suite *FiltererTestSuite) TestStringMatcher_simpleSubMatching() {
	txt := "some-txt"
	parts := []string{"some"}
	suite.True(filterer.StringMather(txt, parts))
}

func (suite *FiltererTestSuite) TestStringMatcher_beginningAndEnd() {
	txt := "some-txt"
	suite.True(filterer.StringMather(txt, []string{"some", "txt"}))
	suite.False(filterer.StringMather(txt, []string{"some", "else"}))
	suite.False(filterer.StringMather(txt, []string{"some", "ome"}))
}

func (suite *FiltererTestSuite) TestStringMatcher_inBetween() {
	txt := "some-thing-txt"
	suite.True(filterer.StringMather(txt, []string{"some", "thing", "txt"}))
	suite.False(filterer.StringMather(txt, []string{"ome", "thing", "txt"}))
	suite.True(filterer.StringMather(txt, []string{"", "thing", "txt"}))
	suite.True(filterer.StringMather(txt, []string{"some", "thing", ""}))
	suite.False(filterer.StringMather(txt, []string{"some", "fail", "txt"}))
	suite.False(filterer.StringMather(txt, []string{"some", "ome", "txt"}))
}

func (suite *FiltererTestSuite) TestFilter_sliceSubstr() {

	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeSlice"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{"*stuff", "some*"},
		ValuesLogicalOperator: query.FilterLogicalOperatorTypeAND,
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeSlice: []string{"wrong-value", "some-stuff"}}))
	suite.True(filter.Run(TestStruct{SomeSlice: []string{"some-stuff"}}))
	suite.False(filter.Run(TestStruct{SomeSlice: []string{"wrong-value", "other-stuff"}}))
	suite.False(filter.Run(TestStruct{SomeSlice: []string{"wrong-value"}}))
}

func (suite *FiltererTestSuite) TestFilter_mapSubstr() {
	options := genOptions([]query.FilterOption{{
		Column:                []string{"SomeMap"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{"key: *"},
		ValuesLogicalOperator: "",
	}})

	filter, err := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)
	suite.NoError(err)
	suite.True(filter.Run(TestStruct{SomeMap: map[string]string{"otherKey": "otherVAl", "key": "val"}}))
	suite.True(filter.Run(TestStruct{SomeMap: map[string]string{"key": "val"}}))
	suite.False(filter.Run(TestStruct{SomeMap: map[string]string{"bad-key": "bad-val"}}))
}

func TestFiltererTestSuite(t *testing.T) {
	suite.Run(t, new(FiltererTestSuite))
}

func genOptions(filters []query.FilterOption) *query.ListOptions {
	options := &query.ListOptions{
		Sorts:      nil,
		Filters:    filters,
		Fields:     nil,
		Pagination: nil,
	}
	return options
}

func MakeList(N int) []TestStruct {
	list := make([]TestStruct, N)
	for i := 0; i < N; i++ {
		list[i].SomeInt = rand.Int()
		list[i].SomeField = random.String(3, "asdfxcvn,m1209")
	}
	return list
}

var table = []struct {
	input int
}{
	{input: 100},
	{input: 1000},
	{input: 10000},
	{input: 100000},
	{input: 1000000},
	// 	{input: 10000000},
}
var maxGen = table[len(table)-1].input

var GenList = MakeList(maxGen)

func BenchmarkFilterer(b *testing.B) {

	options := genOptions([]query.FilterOption{
		{
			Column:                []string{"SomeInt"},
			Operator:              query.FilterOperatorTypeGT,
			Values:                []string{"5"},
			ValuesLogicalOperator: "",
		},
		{
			Column:                []string{"SomeField"},
			Operator:              query.FilterOperatorTypeEQ,
			Values:                []string{"aaa"},
			ValuesLogicalOperator: "",
		},
	})

	filter, _ := filterer.CompileFromQueryListOptions[TestStruct](options.Filters)

	for i := 0; i < b.N; i++ {
		filter.Run(GenList[i%maxGen])
	}

}
