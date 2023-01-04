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
	NewRoutePath = "/config/apps/http/servers/srv0/routes/0"
)

var (
	HostDomainSocket = "//tmp/caddy-admin.sock"
)

type NewRouteRequest struct {
	RouteID                   string
	TargetTunnelHost          string
	TargetTunnelPort          string
	DownstreamProxySubdomain  string
	DownstreamProxyBaseDomain string
}

type API interface {
	AddRoute(ctx context.Context, nrr *NewRouteRequest) (res *http.Response, err error)
	DeleteRoute(ctx context.Context, routeID string) (res *http.Response, err error)
}

func (s *Server) AddRoute(ctx context.Context, nrr *NewRouteRequest) (res *http.Response, err error) {
	body, err := ExecuteTemplate("NRR", NewRouteRequestTemplate, nrr)
	if err != nil {
		return nil, err
	}

	res, err = s.sendRequest(ctx, "PUT", NewRoutePath, body)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Server) DeleteRoute(ctx context.Context, routeID string) (res *http.Response, err error) {
	res, err = s.sendRequest(ctx, "DELETE", makeCaddyResourcePath(routeID), nil)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func makeCaddyResourcePath(in string) (out string) {
	return fmt.Sprintf("/id/%s", in)
}

func (s *Server) sendRequest(ctx context.Context, method string, path string, body []byte) (res *http.Response, err error) {
	var r io.Reader
	if len(body) > 0 {
		r = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, r)
	if err != nil {
		return nil, fmt.Errorf("unable to make new caddy http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send caddy http request: %w", err)
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
