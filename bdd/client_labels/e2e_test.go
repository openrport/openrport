package simple_client_connects_test

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/KonradKuznicki/must"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/cloudradar-monitoring/rport/bdd/helpers"
)

type TagsAndLabels struct {
	Tags   []string          `json:"tags"`
	Labels map[string]string `json:"labels"`
}

type Rsp struct {
	Data []TagsAndLabels `json:"data"`
}

type TagsAndLabelsTestSuite struct {
	suite.Suite
	VariableThatShouldStartAtFive int
	rd                            *exec.Cmd
	rc                            *exec.Cmd
	ctx                           context.Context
}

func (suite *TagsAndLabelsTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctx, cancel := context.WithTimeout(suite.ctx, time.Minute*5)
	defer cancel()
	suite.rd, suite.rc = helpers.StartClientAndServerAndWaitForConnection(ctx, suite.T())
	time.Sleep(time.Millisecond * 100)
	if suite.rc.ProcessState != nil || suite.rd.ProcessState != nil {
		suite.Fail("deamons didn't start")
	}
}

func (suite *TagsAndLabelsTestSuite) TearDownSuite() {
	helpers.LogAndIgnore(suite.rc.Process.Kill())
	helpers.LogAndIgnore(suite.rd.Process.Kill())
}

func (suite *TagsAndLabelsTestSuite) TestClientHasTags() {
	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags"

	expected := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}}}
	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientsCanBeFilteredByTags_findNone() {
	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags&filter[tags]=test"

	expected := []TagsAndLabels{}
	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientsCanBeFilteredByTags_findOne() {

	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags&filter[tags]=vm"

	expected := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}}}
	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientHasLabels() {

	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags,labels"

	expected := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}, Labels: map[string]string{"country": "Germany", "city": "Cologne", "datacenter": "NetCologne GmbH"}}}

	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientHasLabels_findNone() {

	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags,labels&filter[labels]=not_existing"

	expected := []TagsAndLabels{}

	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientHasLabels_findOne() {

	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags,labels&filter[labels]=city_Cologne"

	expected := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}, Labels: map[string]string{"country": "Germany", "city": "Cologne", "datacenter": "NetCologne GmbH"}}}

	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) ExpectAnswer(requestURL string, expected []TagsAndLabels) bool {
	structured := suite.call(requestURL)
	return assert.Equal(suite.T(), structured, Rsp{Data: expected})
}

func (suite *TagsAndLabelsTestSuite) call(requestURL string) Rsp {
	var t *testing.T = suite.T()
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	assert.Nil(t, err)
	req.SetBasicAuth("admin", "foobaz")
	res, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	data2 := string(must.Must(io.ReadAll(res.Body)))
	log.Println(data2)
	data := data2

	var structured Rsp
	must.Must0(json.Unmarshal([]byte(data), &structured))
	return structured
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestTagsAndLabelsTestSuite(t *testing.T) {
	suite.Run(t, new(TagsAndLabelsTestSuite))
}
