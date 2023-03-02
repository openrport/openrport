package chclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

func ConnectionErrorHints(server string, logger *logger.Logger, err error) error {
	switch err.Error() {
	case "Access violation":
		return fmt.Errorf("%s - Check your proxy allows the CONNECT method to the rport server port", err)
	case "Proxy Authentication Required":
		return fmt.Errorf("%s - Check the proxy username and password in rport.conf or add if missing", err)
	case "websocket: bad handshake":
		proxy, allHeaders, checkErr := DetectTransparentProxy(server)
		if checkErr != nil {
			logger.Errorf("error detecting proxy: %s", checkErr)
		}
		if proxy != "" {
			logger.Errorf(proxy)
		}
		if allHeaders != "" {
			logger.Debugf("headers collected while detecting proxy: %s", allHeaders)
		}
		return fmt.Errorf("%s - Server maybe busy. Also check your client credentials AND check for tranparent proxies", err)
	default:
		return err
	}
}

func DetectTransparentProxy(serverURL string) (proxy string, allHeaders string, error error) {
	var client = &http.Client{
		Timeout: time.Second * 10,
	}
	res, err := client.Head(strings.Replace(serverURL, "ws", "http", 1))
	if err != nil {
		return "", "", fmt.Errorf("error on http HEAD request: %s", err)
	}
	proxy = res.Header.Get("Via")
	reqHeadersBytes, err := json.Marshal(res.Header)
	if err != nil {
		return "", "", fmt.Errorf("error on getting headers from http HEAD request: %s", err)
	}
	allHeaders = string(reqHeadersBytes)

	if proxy != "" {
		return fmt.Sprintf("A transparent proxy '%s' seems to interfere your connection", proxy), allHeaders, nil
	}
	return "", allHeaders, nil
}
