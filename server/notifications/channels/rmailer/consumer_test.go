package rmailer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/realvnc-labs/rport/server/notifications/channels/rmailer"
)

func TestNotEscapedMail(t *testing.T) {
	test := "<b>test</b><script>alert('powned!');</script>"
	content, err := rmailer.WrapWithTemplate(test)
	assert.NoError(t, err)
	assert.Contains(t, content, test)
}
