package chshare

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHttpServer(t *testing.T) {
	s := NewHTTPServer(123, nil)

	assert.Equal(t, 123, s.MaxHeaderBytes)
	assert.Equal(t, "", s.certFile)
	assert.Equal(t, "", s.keyFile)
}

func TestNewHttpServerWithTLS(t *testing.T) {
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
	}

	s := NewHTTPServer(123, nil, WithTLS("test.crt", "test.key", tlsConfig))

	assert.Equal(t, 123, s.MaxHeaderBytes)
	assert.Equal(t, "test.crt", s.certFile)
	assert.Equal(t, "test.key", s.keyFile)
}
