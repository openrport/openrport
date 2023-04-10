package client_labels_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
type RspMeta struct {
	Data Metadata `json:"data"`
}

type SaveMetadataTestSuite struct {
	suite.Suite
	serverProcess *exec.Cmd
	clientProcess *exec.Cmd
	ctx           context.Context
	clientId      string
}

func (suite *SaveMetadataTestSuite) SetupSuite() {
	suite.SetupTest()
	suite.ctx = context.Background()
	ctx, cancel := context.WithTimeout(suite.ctx, time.Minute*5)
	defer cancel()
	suite.serverProcess, suite.clientProcess = helpers.StartClientAndServerAndWaitForConnection(ctx, suite.T(), "../../../")
	time.Sleep(time.Millisecond * 100)
	if suite.clientProcess.ProcessState != nil || suite.serverProcess.ProcessState != nil {
		suite.Fail("deamons didn't start")
	}
	suite.clientId = callURL[RspID](suite, "http://localhost:3000/api/v1/clients?fields[clients]=id").Data[0].ID

}

func (suite *SaveMetadataTestSuite) SetupTest() {
	err := os.WriteFile("./client_attributes.json", []byte("{\"tags\":[\"vm\"],\"labels\":{}}"), 0644)
	suite.NoError(err)
}

func (suite *SaveMetadataTestSuite) TearDownSuite() {
	helpers.LogAndIgnore(suite.clientProcess.Process.Kill())
	helpers.LogAndIgnore(suite.serverProcess.Process.Kill())
}

type Metadata struct {
	Tags   []string          `json:"tags"`
	Labels map[string]string `json:"labels"`
}

func (suite *SaveMetadataTestSuite) TestClientHasSomeMetadataPreset() {

	requestURL := "http://localhost:3000/api/v1/clients?fields[clients]=tags,labels&filter[tags]=vm"

	expected := []TagsAndLabels{{Tags: []string{"vm"}, Labels: nil}}

	suite.ExpectAnswer(requestURL, expected)
}

func (suite *SaveMetadataTestSuite) TestClientMetadataIsUpdated() {

	requestURL := fmt.Sprintf("http://localhost:3000/api/v1/clients/%v/metadata", suite.clientId)

	data, err := json.Marshal(Metadata{Tags: []string{"test"}, Labels: map[string]string{"test": "test"}})
	suite.NoError(err)

	checkOperationHttpStatus(suite, requestURL, http.MethodPut, data, http.StatusOK)

	requestURL = "http://localhost:3000/api/v1/clients?fields[clients]=tags,labels&filter[labels]=test:%20test"

	expected := []TagsAndLabels{{Tags: []string{"test"}, Labels: map[string]string{"test": "test"}}}

	suite.ExpectAnswer(requestURL, expected)

}

func (suite *SaveMetadataTestSuite) ExpectAnswer(requestURL string, expected []TagsAndLabels) bool {
	structured := callURL[Rsp](suite, requestURL)
	return suite.Equal(Rsp{Data: expected}, structured)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSaveMetadataTestSuite(t *testing.T) {
	suite.Run(t, new(SaveMetadataTestSuite))
}

func checkOperationHttpStatus(suite *SaveMetadataTestSuite, requestURL string, method string, content []byte, expectedStatus int) {

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(method, requestURL, bytes.NewReader(content))
	suite.NoError(err)
	req.SetBasicAuth("admin", "foobaz")
	res, err := client.Do(req)
	suite.NoError(err)

	suite.Equal(expectedStatus, res.StatusCode)

	rawBody, err := io.ReadAll(res.Body)
	suite.NoError(err)

	body := string(rawBody)
	suite.T().Log(body)

	return
}

func callURL[T any](suite *SaveMetadataTestSuite, requestURL string) T {

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

	var structured T
	suite.NoError(json.Unmarshal([]byte(body), &structured))
	return structured
}
