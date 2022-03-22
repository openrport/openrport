package security

import "crypto/tls"

var TLSConfig = &tls.Config{
	MinVersion:               tls.VersionTLS13,
	CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
	PreferServerCipherSuites: true,
}
