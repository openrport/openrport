package client_labels_test

import (
	"context"
	"encoding/json"
	"fmt"
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
type ID struct {
	ID string `json:"id"`
}

type Rsp struct {
	Data []TagsAndLabels `json:"data"`
}

type RspID struct {
	Data []ID `json:"data"`
}

const apiHost = "http://localhost:4013"

type FailUpdateAttributesTestSuite struct {
	suite.Suite
	serverProcess *exec.Cmd
	clientProcess *exec.Cmd
	ctx           context.Context
	clientID      string
}

func (suite *FailUpdateAttributesTestSuite) SetupSuite() {
	helpers.CleanUp(suite.T(), "./rc-test-resurces", "./rd-test-resources")
	suite.ctx = context.Background()
	ctx, cancel := context.WithTimeout(suite.ctx, time.Minute*5)
	defer cancel()
	suite.serverProcess, suite.clientProcess = helpers.StartClientAndServerAndWaitForConnection(ctx, suite.T(), "../../../")
	time.Sleep(time.Millisecond * 100)
	if suite.clientProcess.ProcessState != nil || suite.serverProcess.ProcessState != nil {
		suite.Fail("deamons didn't start")
	}
	suite.clientID = helpers.CallURL[RspID](&suite.Suite, apiHost+"/api/v1/clients?fields[clients]=id").Data[0].ID

}

func (suite *FailUpdateAttributesTestSuite) TearDownSuite() {
	helpers.LogAndIgnore(suite.clientProcess.Process.Kill())
	helpers.LogAndIgnore(suite.serverProcess.Process.Kill())
}

type Attributes struct {
	Tags   []string          `json:"tags"`
	Labels map[string]string `json:"labels"`
}

func (suite *FailUpdateAttributesTestSuite) TestClientAttributesIsReadOnly() {

	requestURL := fmt.Sprintf(apiHost+"/api/v1/clients/%v/attributes", suite.clientID)

	data, err := json.Marshal(Attributes{Tags: []string{}, Labels: map[string]string{}})
	suite.NoError(err)

	helpers.CheckOperationHTTPStatus(&suite.Suite, requestURL, http.MethodPut, data, http.StatusConflict)
}

func (suite *FailUpdateAttributesTestSuite) ExpectAnswer(requestURL string, expected []TagsAndLabels) bool {
	structured := helpers.CallURL[Rsp](&suite.Suite, requestURL)
	return suite.Equal(Rsp{Data: expected}, structured)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestFailUpdateAttributesTestSuite(t *testing.T) {
	suite.Run(t, new(FailUpdateAttributesTestSuite))
}
