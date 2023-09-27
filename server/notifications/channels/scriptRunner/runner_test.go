package scriptRunner_test

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/openrport/openrport/server/notifications/channels/scriptRunner"
)

var file = "out.json"

type ScriptRunnerTestSuite struct {
	suite.Suite
	timeout time.Duration
	pwd     string
}

func (ts *ScriptRunnerTestSuite) SetupSuite() {
	_ = os.Remove(file)
	ts.timeout = time.Second
	dir, err := os.Getwd()
	ts.NoError(err)
	ts.pwd = dir
}

func (ts *ScriptRunnerTestSuite) TestParamsArePassed() {

	in := "out"

	ts.T().Log("path:", ts.pwd)
	out, err := scriptRunner.RunCancelableScript(context.Background(), ts.pwd, "./test.sh", "out")
	ts.NoError(err)
	ts.Empty(out)

	data, err := os.ReadFile(file)
	ts.NoError(err)
	ts.Equal(in, string(data))
}

func (ts *ScriptRunnerTestSuite) TestScriptTimeout() {

	start := time.Now()
	timeout, cancelFunc := context.WithTimeout(context.Background(), ts.timeout)
	defer cancelFunc()
	out, err := scriptRunner.RunCancelableScript(timeout, ts.pwd, "./test_timeout.sh", "")

	ts.Less(time.Since(start), ts.timeout+time.Second)
	ts.Error(err)
	ts.Empty(out)
}

func (ts *ScriptRunnerTestSuite) TestScriptError() {
	out, err := scriptRunner.RunCancelableScript(context.Background(), ts.pwd, "./test_error.sh", "")

	ts.Error(err)
	ts.Empty(out)
}

func (ts *ScriptRunnerTestSuite) TestScriptDir() {
	out, err := scriptRunner.RunCancelableScript(context.Background(), "/tmp", path.Join(ts.pwd, "./test_pwd.sh"), "")
	ts.NoError(err)
	ts.NotEmpty(out)

	ts.NotEqual(strings.Trim(out, "\n \t"), ts.pwd)

	ts.T().Log("paths", ts.pwd, out)
}

func (ts *ScriptRunnerTestSuite) TestScriptStdError() {
	out, err := scriptRunner.RunCancelableScript(context.Background(), ts.pwd, "./test_stderr.sh", "")

	ts.Error(err)
	ts.Empty(out)
}

func TestScriptRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptRunnerTestSuite))
}
