package formatter_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/share/dynops/formatter"
)

type FormatTestSuite struct {
	suite.Suite
}

func (suite *FormatTestSuite) SetupTest() {
	// setup
}

func (suite *FormatTestSuite) format(object any, fields []string) (map[string]interface{}, error) {
	ft := formatter.NewFormatter(object)
	translator, err := ft.NewTranslator(fields)
	if err != nil {
		return nil, err
	}
	return translator.Format(object), nil
}

func (suite *FormatTestSuite) TestFormatNoFieldsShouldGiveEmptyObjet() {
	formatted, _ := suite.format(struct{}{}, []string{})
	suite.Equal(map[string]interface{}{}, formatted)
}

func (suite *FormatTestSuite) TestFormatNonExistentFieldReturnError() {
	_, err := suite.format(struct{}{}, []string{"test-field"})
	suite.ErrorContains(err, "requested field does not exist: test-field")
}

type ForTest struct {
	SomeField    string
	OtherField   string `json:"other"`
	IgnoredField string `json:"-"`
}

func (suite *FormatTestSuite) TestFormatRequestedExistentFieldIsPresentInTheResult() {
	formatted, err := suite.format(ForTest{SomeField: "test-value"}, []string{"SomeField"})
	suite.Nil(err)
	suite.Equal(map[string]interface{}{"SomeField": "test-value"}, formatted)
}

func (suite *FormatTestSuite) TestFormatUseJsonTagIfPresent() {
	formatted, err := suite.format(ForTest{SomeField: "test-value", OtherField: "val"}, []string{"SomeField", "other"})
	suite.Nil(err)
	suite.Equal(map[string]interface{}{"SomeField": "test-value", "other": "val"}, formatted)
}

func (suite *FormatTestSuite) TestFormatErrorOnFieldsThatAreIgnoredInJsonTag() {
	formatted, err := suite.format(ForTest{}, []string{"SomeField", "IgnoredField"})
	suite.Nil(formatted)
	suite.ErrorContains(err, "requested field does not exist: IgnoredField")
}

func (suite *FormatTestSuite) TestFormatSerializesNicelyToJson() {
	formatted, err := suite.format(ForTest{OtherField: "val"}, []string{"other"})
	suite.Nil(err)

	out, _ := json.Marshal(formatted)

	suite.Equal("{\"other\":\"val\"}", string(out))
}

func (suite *FormatTestSuite) TestFormatSerializesPointersNicelyToJson() {
	formatted, err := suite.format(&ForTest{OtherField: "val"}, []string{"other"})
	suite.Nil(err)

	out, _ := json.Marshal(formatted)

	suite.Equal("{\"other\":\"val\"}", string(out))
}

type Tt struct {
	A string
}

func (suite *FormatTestSuite) TestAsdf() {
	var t *Tt
	suite.Equal(map[string]int{}, formatter.BuildTranslationTable(t))
}

func TestFormatTestSuite(t *testing.T) {
	suite.Run(t, new(FormatTestSuite))
}
