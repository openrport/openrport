package simple_client_connects_test

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/bdd/helpers"
)

func TestClientConnects(t *testing.T) {

	ctx := context.Background()

	rd, rc := helpers.StartClientAndServerAndWaitForConnection(ctx, t)

	defer func() {
		helpers.Yolo(rd.Process.Kill())
		helpers.Yolo(rc.Process.Kill())
	}()

	assertProcessiesAreNotDead(t, rd, rc)
}

func assertProcessiesAreNotDead(t *testing.T, rd *exec.Cmd, rc *exec.Cmd) {
	assert.Nil(t, rd.ProcessState)
	assert.Nil(t, rc.ProcessState)
}
