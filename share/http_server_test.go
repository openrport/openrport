package chshare

import (
	"crypto/tls"
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
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	s := NewHTTPServer(123, WithTLS("test.crt", "test.key", tlsConfig))

	assert.Equal(t, 123, s.MaxHeaderBytes)
	assert.Equal(t, "test.crt", s.certFile)
	assert.Equal(t, "test.key", s.keyFile)
}
