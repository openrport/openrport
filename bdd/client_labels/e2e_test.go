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
	Tags   []string `json:"tags"`
	Labels []string `json:"labels"`
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
	suite.rd, suite.rc = helpers.StartClientAndServerAndWaitForConnection(suite.ctx, suite.T())
	time.Sleep(time.Millisecond * 100)
	if suite.rc.ProcessState != nil || suite.rd.ProcessState != nil {
		suite.Fail("deamons didn't start")
	}
}

func (suite *TagsAndLabelsTestSuite) TearDownSuite() {
	suite.rc.Process.Kill()
	suite.rd.Process.Kill()
}

func (suite *TagsAndLabelsTestSuite) TestClientHasTags() {
	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags"

	content := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}}}
	suite.ExpectAnswer(requestURL, content)
}

func (suite *TagsAndLabelsTestSuite) TestClientsCanBeFilteredByTags_findNone() {
	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags&filter[tags]=test"

	content := []TagsAndLabels{}
	suite.ExpectAnswer(requestURL, content)
}

func (suite *TagsAndLabelsTestSuite) TestClientsCanBeFilteredByTags_findOne() {
	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags&filter[tags]=vm"

	content := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}}}
	suite.ExpectAnswer(requestURL, content)
}

func (suite *TagsAndLabelsTestSuite) TestClientHasLabels() {
	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags,labels"

	content := []TagsAndLabels{{Tags: []string{"task-vm", "vm"}}}

	suite.ExpectAnswer(requestURL, content)
}

func (suite *TagsAndLabelsTestSuite) ExpectAnswer(requestURL string, content []TagsAndLabels) bool {
	structured := suite.call(requestURL)
	return assert.Equal(suite.T(), structured, Rsp{Data: content})
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
