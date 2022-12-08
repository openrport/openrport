package certs_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/certs"
)

const (
	testCertsPath = "../../testdata/certs"
)

func TestShouldDoSomething(t *testing.T) {
	path := testCertsPath + "/" + "rptest.io.crt"

	certBytes, err := os.ReadFile(path)
	require.NoError(t, err)

	cert, err := certs.ParseCertificate(certBytes)
	require.NoError(t, err)

	fmt.Printf("cert.DNSNames = %+v\n", cert.DNSNames)
}
