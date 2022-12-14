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
	path := testCertsPath + "/" + "rpdev.lan.crt"

	certBytes, err := os.ReadFile(path)
	require.NoError(t, err)

	cert, err := certs.ParseCertificate(certBytes)
	require.NoError(t, err)

	fmt.Printf("cert.DNSNames = %+v\n", cert.DNSNames)
	// fmt.Printf("cert.Extensions = %+v\n", cert.Extensions)
	// exts := cert.Extensions
	// ext := exts[0]
	// fmt.Printf("ext.Value = %+v\n", ext.Value)
	// var v1 asn1.RawValue
	// _, err = asn1.Unmarshal(ext.Value, &v1)
	// require.NoError(t, err)
	// fmt.Printf("v1 = %+v\n", v1)
	// var v2 asn1.RawValue
	// _, err = asn1.Unmarshal(v1.Bytes, &v2)
	// require.NoError(t, err)
	// fmt.Printf("v2 = %+v\n", v2)
	// fmt.Printf("v2.Bytes = %+v\n", string(v2.Bytes))
}
