package chshare

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHttpServer(t *testing.T) {
	s := NewHTTPServer(123)

	assert.Equal(t, 123, s.MaxHeaderBytes)
	assert.Equal(t, "", s.certFile)
	assert.Equal(t, "", s.keyFile)
}

func TestNewHttpServerWithTLS(t *testing.T) {
	s := NewHTTPServer(123, WithTLS("test.crt", "test.key"))

	assert.Equal(t, 123, s.MaxHeaderBytes)
	assert.Equal(t, "test.crt", s.certFile)
	assert.Equal(t, "test.key", s.keyFile)
}
