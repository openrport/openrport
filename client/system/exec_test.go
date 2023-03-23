package system

import (
	"context"
	"os"
	"testing"

	"github.com/realvnc-labs/rport/share/logger"

	"github.com/stretchr/testify/assert"
)

type cmdBuildTestCase struct {
	name               string
	interpreter        string
	wantCmdStr         string
	command            string
	partialMatch       bool
	boolHasShebang     bool
	interpreterAliases map[string]string
}

func TestBuildCmd(t *testing.T) {
	var testLog = logger.NewLogger("client-system", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	cmdExecutor := NewCmdExecutor(testLog)
	for _, tc := range getCmdBuildTestcases() {
		t.Run(tc.name, func(t *testing.T) {
			interpreter := Interpreter{
				InterpreterNameFromInput: tc.interpreter,
				InterpreterAliases:       tc.interpreterAliases,
			}
			execCtx := &CmdExecutorContext{
				Interpreter: interpreter,
				HasShebang:  tc.boolHasShebang,
				Command:     tc.command,
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
