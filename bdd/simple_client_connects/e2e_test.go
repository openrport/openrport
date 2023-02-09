package simple_client_connects_test

import (
	"github.com/cloudradar-monitoring/rport/bdd/helpers"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientConnects(t *testing.T) {

	rd, rdOutChan, _ := helpers.Run(t, "", "../../cmd/rportd/main.go")
	defer func() {
		rd.Process.Kill()
	}()

	err := helpers.WaitForText(rdOutChan, "API Listening") // wait for server to initialize and boot
	assert.Nil(t, err)

	rc, rcOutChan, _ := helpers.Run(t, "", "../../cmd/rport/main.go")
	defer func() {
		rc.Process.Kill()
	}()

	err = helpers.WaitForText(rcOutChan, "info: client: Connected") // wait for client to connect
	assert.Nil(t, err)

}
