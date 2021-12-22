package system

import (
	"os"
	"testing"

	"github.com/cloudradar-monitoring/rport/share/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLog = logger.NewLogger("client-system", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

type interpreterTestCase struct {
	name               string
	interpreter        string
	wantInterpreter    string
	wantErrContains    string
	boolHasShebang     bool
	interpreterAliases map[string]string
}

func TestGetInterpreter(t *testing.T) {
	cmdExecutor := NewCmdExecutor(testLog)
	for _, tc := range getInterpreterTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			execCtx := &CmdExecutorContext{
				Interpreter:        tc.interpreter,
				HasShebang:         tc.boolHasShebang,
				InterpreterAliases: tc.interpreterAliases,
			}
			// when
			gotInterpreter, gotErr := cmdExecutor.getInterpreter(execCtx)

			// then
			if len(tc.wantErrContains) > 0 {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, gotErr)
				assert.Equal(t, tc.wantInterpreter, gotInterpreter)
			}
		})
	}
}
