package security

import "crypto/tls"

func TLSConfig(configTLSMin string) *tls.Config {
	tlsMin := uint16(tls.VersionTLS13)
	switch configTLSMin {
	case "1.2":
		tlsMin = tls.VersionTLS12
	}

	// #nosec G402 -- disables G402: TLS MinVersion too low. (gosec)
	var TLSConfig = &tls.Config{
		MinVersion:               tlsMin,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
	}
	return TLSConfig
}
