package simple_client_connects_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openrport/openrport/bdd/helpers"
)

func TestClientConnects(t *testing.T) {
	helpers.CleanUp(t, "./rc-test-resurces", "./rd-test-resources")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	serverProcess, clientProcess := helpers.StartClientAndServerAndWaitForConnection(ctx, t, helpers.FindProjectRoot(t))

	defer func() {
		helpers.LogAndIgnore(serverProcess.Process.Kill())
		helpers.LogAndIgnore(clientProcess.Process.Kill())
	}()

	assertProcessiesAreNotDead(t, serverProcess, clientProcess)
}

func assertProcessiesAreNotDead(t *testing.T, serverProcess *exec.Cmd, clientProcess *exec.Cmd) {
	assert.Nil(t, serverProcess.ProcessState)
	assert.Nil(t, clientProcess.ProcessState)
}
