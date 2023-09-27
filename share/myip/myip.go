package myip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/openrport/openrport/share/models"
)

const (
	ipV4      = "tcp4"
	ipV6      = "tcp6"
	userAgent = ""
)

type APIResponse struct {
	IP string `json:"ip"`
}

func GetMyIPs(ctx context.Context, apiURL string) (ips *models.IPAddresses, err error) {
	ips = &models.IPAddresses{}

	IPv4, err := getMyIP(ctx, apiURL, ipV4)
	if err != nil {
		ips.Error = err.Error()
		return ips, err
	}
	ips.IPv4 = IPv4

	IPv6, err := getMyIP(ctx, apiURL, ipV6)
	if err != nil {
		ips.Error = err.Error()
		return ips, err
	}
	ips.IPv6 = IPv6

	return ips, nil
}

func getMyIP(ctx context.Context, apiURL string, netTransport string) (ip string, err error) {
	var zeroDialer net.Dialer
	var httpClient = &http.Client{
		Timeout: 2 * time.Second,
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return zeroDialer.DialContext(ctx, netTransport, addr)
	}
	httpClient.Transport = transport

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("request to %s failed: %s", apiURL, err)
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("request to %s failed with status %d: %s", apiURL, res.StatusCode, b)
	}
	var a APIResponse
	err = json.NewDecoder(res.Body).Decode(&a)
	if err != nil {
		return "", err
	}
	return a.IP, nil
}
