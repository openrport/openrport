package system

import (
	"context"
	"os"
	"testing"

	"github.com/cloudradar-monitoring/rport/share/logger"

	"github.com/stretchr/testify/assert"
)

var testLog = logger.NewLogger("client-system", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

type interpreterTestCase struct {
	name               string
	interpreter        string
	wantCmdStr         string
	command            string
	partialMatch       bool
	boolHasShebang     bool
	interpreterAliases map[string]string
}

func TestBuildCmd(t *testing.T) {
	cmdExecutor := NewCmdExecutor(testLog)
	for _, tc := range getInterpreterTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			execCtx := &CmdExecutorContext{
				Interpreter:        tc.interpreter,
				HasShebang:         tc.boolHasShebang,
				InterpreterAliases: tc.interpreterAliases,
				Command:            tc.command,
			}
			// when
			cmd := cmdExecutor.New(context.Background(), execCtx)
			if tc.partialMatch {
				assert.Contains(t, cmd.String(), tc.wantCmdStr)
			} else {
				assert.Equal(t, tc.wantCmdStr, cmd.String())
			}
		})
	}
}
