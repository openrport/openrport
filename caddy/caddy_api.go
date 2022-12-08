package caddy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
)

const (
	HostDomainSocket = "//tmp/caddy-admin.sock"
	NewRoutePath     = "/config/apps/http/servers/srv0/routes/0"
)

func AddRoute(ctx context.Context, nrr *NewRouteRequest) (res *http.Response, err error) {
	body, err := ExecuteTemplate("NRR", NewRouteRequestTemplate, nrr)
	if err != nil {
		return nil, err
	}

	res, err = SendRequest(ctx, "PUT", NewRoutePath, body)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func DeleteRoute(ctx context.Context, routeID string) (res *http.Response, err error) {
	res, err = SendRequest(ctx, "DELETE", makeCaddyResourcePath(routeID), nil)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func makeCaddyResourcePath(in string) (out string) {
	return fmt.Sprintf("/id/%s", in)
}

func SendRequest(ctx context.Context, method string, path string, body []byte) (res *http.Response, err error) {
	httpClient := newHTTPDomainSocketClient()

	var r io.Reader
	if len(body) > 0 {
		r = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, "http://unix"+path, r)
	if err != nil {
		return nil, fmt.Errorf("unable to make new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err = httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send request: %w", err)
	}

	return res, err
}

func newHTTPDomainSocketClient() (httpClient http.Client) {
	httpClient = http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", HostDomainSocket)
			},
		},
	}
	return httpClient
}
