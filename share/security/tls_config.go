package security

import "crypto/tls"

func TLSConfig(whichtls uint16) *tls.Config {
	var TLSConfig = &tls.Config{
		MinVersion:               whichtls,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
	}
	return TLSConfig
}
