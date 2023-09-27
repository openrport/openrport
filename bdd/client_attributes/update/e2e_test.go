package client_labels_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/openrport/openrport/bdd/helpers"
)

type TagsAndLabels struct {
	Tags   []string          `json:"tags"`
	Labels map[string]string `json:"labels"`
}
type ID struct {
	ID string `json:"id"`
}

type Rsp struct {
	Data []TagsAndLabels `json:"data"`
}

type RspID struct {
	Data []ID `json:"data"`
}

const apiHost = "http://localhost:4011"

type UpdateAttributesTestSuite struct {
	suite.Suite
	serverProcess *exec.Cmd
	clientProcess *exec.Cmd
	ctx           context.Context
	clientID      string
}

func (suite *UpdateAttributesTestSuite) SetupTest() {
	helpers.CleanUp(suite.T(), "./rc-test-resurces", "./rd-test-resources")
	err := os.WriteFile("./client_attributes.json", []byte("{\"tags\":[\"vm\"],\"labels\":{}}"), 0600)
	suite.NoError(err)
	suite.ctx = context.Background()
	ctx, cancel := context.WithTimeout(suite.ctx, time.Minute*5)
	defer cancel()
	suite.serverProcess, suite.clientProcess = helpers.StartClientAndServerAndWaitForConnection(ctx, suite.T(), helpers.FindProjectRoot(suite.T()))

	suite.clientID = helpers.CallURL[RspID](&suite.Suite, apiHost+"/api/v1/clients?fields[clients]=id").Data[0].ID
}

func (suite *UpdateAttributesTestSuite) TearDownTest() {
	helpers.LogAndIgnore(suite.clientProcess.Process.Kill())
	helpers.LogAndIgnore(suite.serverProcess.Process.Kill())
	log.Println("done")
}

type Attributes struct {
	Tags   []string          `json:"tags"`
	Labels map[string]string `json:"labels"`
}

func (suite *UpdateAttributesTestSuite) TestClientAttributesIsUpdated() {

	requestURL := fmt.Sprintf(apiHost+"/api/v1/clients/%v/attributes", suite.clientID)

	data, err := json.Marshal(Attributes{Tags: []string{"test"}, Labels: map[string]string{"test": "test"}})
	suite.NoError(err)

	helpers.CheckOperationHTTPStatus(&suite.Suite, requestURL, http.MethodPut, data, http.StatusOK)

	requestURL = apiHost + "/api/v1/clients?fields[clients]=tags,labels&filter[labels]=test:%20test"

	expected := []TagsAndLabels{{Tags: []string{"test"}, Labels: map[string]string{"test": "test"}}}

	suite.ExpectAnswer(requestURL, expected)

	suite.TearDownTest()
}

func (suite *UpdateAttributesTestSuite) ExpectAnswer(requestURL string, expected []TagsAndLabels) bool {
	structured := helpers.CallURL[Rsp](&suite.Suite, requestURL)
	return suite.Equal(Rsp{Data: expected}, structured)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestUpdateAttributesTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateAttributesTestSuite))
}
