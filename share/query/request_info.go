package query

import "net/http"

type RequestInfo struct {
	URL string
}

func ParseRequestInfo(req *http.Request) *RequestInfo {
	r := &RequestInfo{
		URL: req.Host + req.URL.Path,
	}
	if req.TLS == nil {
		r.URL = "http://" + r.URL
	} else {
		r.URL = "https://" + r.URL
	}

	return r
}
