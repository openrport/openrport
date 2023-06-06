package scriptRunner_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/server/notifications/channels/scriptRunner"
	"github.com/realvnc-labs/rport/share/simpleops"
)

var file = "out.json"

type ScriptRunnerTestSuite struct {
	suite.Suite
	runner  scriptRunner.ScriptRunner
	timeout time.Duration
}

func (ts *ScriptRunnerTestSuite) SetupSuite() {
	_ = os.Remove(file)
	ts.timeout = time.Second
	ts.runner = scriptRunner.NewScriptRunner(ts.timeout)
}

type ScriptIO struct {
	Subject    string   `json:"subject"`
	Recipients []string `json:"recipients"`
	Content    string   `json:"content"`
}

func (ts *ScriptRunnerTestSuite) TestParamsArePassed() {

	in := ScriptIO{
		Subject:    "test-subject",
		Recipients: []string{"r1@example.com", "somethin323-55@test.co"},
		Content:    "test-content",
	}

	err := ts.runner.Run("./test.sh", in.Recipients, in.Subject, in.Content)
	ts.NoError(err)

	out, err := simpleops.ReadJSONFileIntoStruct[ScriptIO](file)
	ts.NoError(err)
	ts.Equal(in, out)
}

func (ts *ScriptRunnerTestSuite) TestScriptTimeout() {
	in := ScriptIO{}

	start := time.Now()
	err := ts.runner.Run("./test_timeout.sh", in.Recipients, in.Subject, in.Content)
	ts.Less(time.Now().Sub(start), ts.timeout+time.Second)
	ts.Error(err)
}

func (ts *ScriptRunnerTestSuite) TestScriptError() {
	in := ScriptIO{}

	err := ts.runner.Run("./test_error.sh", in.Recipients, in.Subject, in.Content)

	ts.Error(err)
}

func (ts *ScriptRunnerTestSuite) TestScriptStdError() {
	in := ScriptIO{}

	err := ts.runner.Run("./test_stderr.sh", in.Recipients, in.Subject, in.Content)

	ts.Error(err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestScriptRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptRunnerTestSuite))
}
