package client_labels_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/bdd/helpers"
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
	serverProcess *exec.Cmd
	clientProcess *exec.Cmd
	ctx           context.Context
}

func (suite *TagsAndLabelsTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctx, cancel := context.WithTimeout(suite.ctx, time.Minute*5)
	defer cancel()
	suite.serverProcess, suite.clientProcess = helpers.StartClientAndServerAndWaitForConnection(ctx, suite.T())
	time.Sleep(time.Millisecond * 100)
	if suite.clientProcess.ProcessState != nil || suite.serverProcess.ProcessState != nil {
		suite.Fail("deamons didn't start")
	}
}

func (suite *TagsAndLabelsTestSuite) TearDownSuite() {
	helpers.LogAndIgnore(suite.clientProcess.Process.Kill())
	helpers.LogAndIgnore(suite.serverProcess.Process.Kill())
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

	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags,labels&filter[labels]=city:%20Cologne"

	expected := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}, Labels: map[string]string{"country": "Germany", "city": "Cologne", "datacenter": "NetCologne GmbH"}}}

	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) ExpectAnswer(requestURL string, expected []TagsAndLabels) bool {
	structured := suite.callURL(requestURL)
	return suite.Equal(structured, Rsp{Data: expected})
}

func (suite *TagsAndLabelsTestSuite) callURL(requestURL string) Rsp {

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	suite.NoError(err)
	req.SetBasicAuth("admin", "foobaz")
	res, err := client.Do(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, res.StatusCode)

	rawBody, err := io.ReadAll(res.Body)
	suite.NoError(err)

	body := string(rawBody)
	suite.T().Log(body)

	var structured Rsp
	suite.NoError(json.Unmarshal([]byte(body), &structured))
	return structured
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestTagsAndLabelsTestSuite(t *testing.T) {
	suite.Run(t, new(TagsAndLabelsTestSuite))
}
