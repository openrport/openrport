package filter_test

import (
	"context"
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

const apiHost = "http://localhost:4012"

type TagsAndLabelsTestSuite struct {
	suite.Suite
	serverProcess *exec.Cmd
	clientProcess *exec.Cmd
	ctx           context.Context
}

func (suite *TagsAndLabelsTestSuite) SetupSuite() {
	helpers.CleanUp(suite.T(), "./rc-test-resurces", "./rd-test-resources")
	suite.ctx = context.Background()
	ctx, cancel := context.WithTimeout(suite.ctx, time.Minute*5)
	defer cancel()
	suite.serverProcess, suite.clientProcess = helpers.StartClientAndServerAndWaitForConnection(ctx, suite.T(), "../../../")
	time.Sleep(time.Millisecond * 100)
	if suite.clientProcess.ProcessState != nil || suite.serverProcess.ProcessState != nil {
		suite.Fail("daemons didn't start")
	}
}

func (suite *TagsAndLabelsTestSuite) TearDownSuite() {
	helpers.LogAndIgnore(suite.clientProcess.Process.Kill())
	helpers.LogAndIgnore(suite.serverProcess.Process.Kill())
}

func (suite *TagsAndLabelsTestSuite) TestClientHasTags() {
	requestURL := apiHost + "/api/v1/clients?fields[clients]=tags"

	expected := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}}}
	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientsCanBeFilteredByTags_findNone() {
	requestURL := apiHost + "/api/v1/clients?fields[clients]=tags&filter[tags]=test"

	expected := []TagsAndLabels{}
	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientsCanBeFilteredByTags_findOne() {

	requestURL := apiHost + "/api/v1/clients?fields[clients]=tags&filter[tags]=vm"

	expected := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}}}
	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientHasLabels() {

	requestURL := apiHost + "/api/v1/clients?fields[clients]=tags,labels"

	expected := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}, Labels: map[string]string{"country": "Germany", "city": "Cologne", "datacenter": "NetCologne GmbH"}}}

	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientHasLabels_findNone() {

	requestURL := apiHost + "/api/v1/clients?fields[clients]=tags,labels&filter[labels]=not_existing"

	expected := []TagsAndLabels{}

	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) TestClientHasLabels_findOne() {

	requestURL := apiHost + "/api/v1/clients?fields[clients]=tags,labels&filter[labels]=city:%20Cologne"

	expected := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}, Labels: map[string]string{"country": "Germany", "city": "Cologne", "datacenter": "NetCologne GmbH"}}}

	suite.ExpectAnswer(requestURL, expected)
}

func (suite *TagsAndLabelsTestSuite) ExpectAnswer(requestURL string, expected []TagsAndLabels) bool {
	structured := helpers.CallURL[Rsp](&suite.Suite, requestURL)
	return suite.Equal(Rsp{Data: expected}, structured)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestTagsAndLabelsTestSuite(t *testing.T) {
	helpers.CleanUp(t, "./rc-test-resurces", "./rd-test-resources")
	suite.Run(t, new(TagsAndLabelsTestSuite))
}
