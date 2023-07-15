package scriptRunner_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/server/notifications/channels/scriptRunner"
)

var file = "out.json"

type ScriptRunnerTestSuite struct {
	suite.Suite
	timeout time.Duration
}

func (ts *ScriptRunnerTestSuite) SetupSuite() {
	_ = os.Remove(file)
	ts.timeout = time.Second
}

func (ts *ScriptRunnerTestSuite) TestParamsArePassed() {

	in := "out"

	err := scriptRunner.RunCancelableScript(context.Background(), "./test.sh", "out")
	ts.NoError(err)

	data, err := os.ReadFile(file)
	ts.NoError(err)
	ts.Equal(in, string(data))
}

func (ts *ScriptRunnerTestSuite) TestScriptTimeout() {

	start := time.Now()
	timeout, cancelFunc := context.WithTimeout(context.Background(), ts.timeout)
	defer cancelFunc()
	err := scriptRunner.RunCancelableScript(timeout, "./test_timeout.sh", "")

	ts.Less(time.Since(start), ts.timeout+time.Second)
	ts.Error(err)
}

func (ts *ScriptRunnerTestSuite) TestScriptError() {
	err := scriptRunner.RunCancelableScript(context.Background(), "./test_error.sh", "")

	ts.Error(err)
}

func (ts *ScriptRunnerTestSuite) TestScriptStdError() {
	err := scriptRunner.RunCancelableScript(context.Background(), "./test_stderr.sh", "")

	ts.Error(err)
}

func TestScriptRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptRunnerTestSuite))
}
