package chshare_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	chshare "github.com/realvnc-labs/rport/share"
)

const expectedKey = "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIEGV9cYbglKJqaqRWLU5I8RW2Urc4knsWpnfl54N5PeLoAoGCCqGSM49\nAwEHoUQDQgAEM4X+JcCAAGUcAS/cfQ5iF8evBjwpJDhgbJZ/lQYcA6JX34WkjI4N\nalmivx1pS7D4tq7M2u6JKDK6Ff7kjWxBZQ==\n-----END EC PRIVATE KEY-----\n"
const seed = "Z4mQqmkDnexNNR4giNU3Me"

func NativeGenerateKey(seed string) ([]byte, error) {
	var r io.Reader
	if seed == "" {
		r = rand.Reader
	} else {
		r = chshare.NewDetermRand([]byte(seed))
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), r)
	if err != nil {
		return nil, err
	}
	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("Unable to marshal ECDSA private key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}), nil
}

// this test is left to check compatibility with go 1.19
func TestKey(t *testing.T) {
	if runtime.Version() > "go1.19.10" {
		t.Logf("for go version: %v this test is disabled", runtime.Version())
		t.Skip()
	}

	key, err := NativeGenerateKey(seed)
	assert.NoError(t, err)
	assert.Equal(t, expectedKey, string(key))
}

func TestGenerateKey(t *testing.T) {
	compatibleKey, err := chshare.GenerateKey(seed)
	assert.NoError(t, err)
	assert.Equal(t, expectedKey, string(compatibleKey))
}

func TestEquality(t *testing.T) {
	key, err := chshare.GenerateKey(seed)
	assert.NoError(t, err)
	compatibleKey, err := NativeGenerateKey(seed)
	assert.NoError(t, err)

	if runtime.Version() < "go1.20" && runtime.Version() > "go1.19" {
		t.Logf("for go version: %v keys generations should be identical", runtime.Version())
		assert.Equal(t, string(key), string(compatibleKey))
	} else {
		t.Logf("for go version: %v keys generations should be different", runtime.Version())
		assert.NotEqual(t, string(key), string(compatibleKey))
	}

}
