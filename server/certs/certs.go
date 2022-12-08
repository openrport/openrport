package certs

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func ParseCertificate(certBytes []byte) (cert *x509.Certificate, err error) {
	raw := certBytes
	for {
		block, rest := pem.Decode(raw)
		if block == nil {
			break
		}

		if block.Type == "CERTIFICATE" {
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			return cert, nil
		}

		raw = rest
	}

	return nil, errors.New("no cert block found")
}
