package clients

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTunnelACL(t *testing.T) {
	acl, err := ParseTunnelACL(LocalHost)
	assert.Nil(t, err)
	assert.NotNil(t, acl)
	assert.Equal(t, acl.AllowedIPs[0].String(), LocalHost+"/32")
}
