package security

import "crypto/tls"

func TLSConfig(whichtls uint16) *tls.Config {
	// #nosec G402 -- disables G402: TLS MinVersion too low. (gosec)
	var TLSConfig = &tls.Config{
		MinVersion:               whichtls,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
	}
	return TLSConfig
}
