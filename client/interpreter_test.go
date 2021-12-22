package chclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type interpreterTestCase struct {
	name               string
	interpreter        string
	wantInterpreter    string
	wantErrContains    string
	boolHasShebang     bool
	interpreterAliases map[string]string
}

func TestGetInterpreter(t *testing.T) {
	for _, tc := range getInterpreterTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			// when
			interpreterInput := interpreterProviderInput{
				name:       tc.interpreter,
				hasShebang: tc.boolHasShebang,
				aliasesMap: tc.interpreterAliases,
			}
			gotInterpreter, gotErr := getInterpreter(interpreterInput)

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
