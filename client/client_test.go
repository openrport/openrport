package chclient

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/stretchr/testify/assert"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func TestCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Foo") != "Bar" {
			t.Fatal("expected header Foo to be 'Bar'")
		}
	}))
	// Close the server when test finishes
	defer server.Close()

	config := Config{
		Client: ClientConfig{
			Fingerprint: "",
			Auth:        "",
			Server:      server.URL,
			Remotes:     []string{"192.168.0.5:3000:google.com:80"},
		},
		Connection: ConnectionConfig{
			KeepAlive:        time.Second,
			MaxRetryCount:    0,
			MaxRetryInterval: time.Second,
			HeadersRaw:       []string{"Foo: Bar"},
		},
	}
	err := config.ParseAndValidate()
	if err != nil {
		log.Fatal(err)
	}
	c := NewClient(&config)
	if err = c.Run(); err != nil {
		log.Fatal(err)
	}
}

func TestConnectionRequest(t *testing.T) {
	remote1 := &chshare.Remote{
		LocalHost:  "test-local",
		LocalPort:  "1234",
		RemoteHost: "test-remote",
		RemotePort: "2345",
	}
	remote2 := &chshare.Remote{
		LocalHost:  "test-local-2",
		LocalPort:  "2234",
		RemoteHost: "test-remote-2",
		RemotePort: "3345",
	}
	config := &Config{
		Client: ClientConfig{
			ID:      "test-client-id",
			Name:    "test-name",
			Tags:    []string{"tag1", "tag2"},
			remotes: []*chshare.Remote{remote1, remote2},
		},
	}
	interfaceAddrs := []net.Addr{
		&net.IPAddr{
			IP: net.ParseIP("192.0.2.1"),
		},
		&net.IPAddr{
			IP: net.ParseIP("2001:db8::1"),
		},
		&net.IPNet{
			IP: net.ParseIP("192.0.2.2"),
		},
		&net.IPNet{
			IP: net.ParseIP("2001:db8::2"),
		},
	}

	testCases := []struct {
		Name                      string
		SystemInfo                SystemInfo
		ExpectedConnectionRequest *chshare.ConnectionRequest
	}{
		{
			Name: "no errors",
			SystemInfo: &mockSystemInfo{
				ReturnHostname: "test-hostname",
				ReturnUname:    "test-uname",
				ReturnHostInfo: &host.InfoStat{
					OS:             "test-os",
					PlatformFamily: "test-family",
				},
				ReturnInterfaceAddrs: interfaceAddrs,
				ReturnGoArch:         "test-arch",
			},
			ExpectedConnectionRequest: &chshare.ConnectionRequest{
				Version:  "0.0.0-src",
				ID:       "test-client-id",
				Name:     "test-name",
				Tags:     []string{"tag1", "tag2"},
				Remotes:  []*chshare.Remote{remote1, remote2},
				OS:       "test-uname",
				OSArch:   "test-arch",
				OSFamily: "test-family",
				OSKernel: "test-os",
				Hostname: "test-hostname",
				IPv4:     []string{"192.0.2.1", "192.0.2.2"},
				IPv6:     []string{"2001:db8::1", "2001:db8::2"},
			},
		}, {
			Name: "windows, no errors",
			SystemInfo: &mockSystemInfo{
				ReturnHostname: "test-hostname",
				ReturnUname:    "test-uname",
				ReturnHostInfo: &host.InfoStat{
					OS:              "windows",
					Platform:        "test-platform",
					PlatformVersion: "123",
					PlatformFamily:  "test-family",
				},
				ReturnInterfaceAddrs: interfaceAddrs,
				ReturnGoArch:         "test-arch",
			},
			ExpectedConnectionRequest: &chshare.ConnectionRequest{
				Version:  "0.0.0-src",
				ID:       "test-client-id",
				Name:     "test-name",
				Tags:     []string{"tag1", "tag2"},
				Remotes:  []*chshare.Remote{remote1, remote2},
				OS:       "test-platform 123 test-family",
				OSArch:   "test-arch",
				OSFamily: "test-family",
				OSKernel: "windows",
				Hostname: "test-hostname",
				IPv4:     []string{"192.0.2.1", "192.0.2.2"},
				IPv6:     []string{"2001:db8::1", "2001:db8::2"},
			},
		}, {
			Name: "all errors",
			SystemInfo: &mockSystemInfo{
				ReturnHostnameError:       errors.New("test error"),
				ReturnUnameError:          errors.New("test error"),
				ReturnHostInfoError:       errors.New("test error"),
				ReturnInterfaceAddrsError: errors.New("test error"),
				ReturnGoArch:              "test-arch",
			},
			ExpectedConnectionRequest: &chshare.ConnectionRequest{
				Version:  "0.0.0-src",
				ID:       "test-client-id",
				Name:     "test-name",
				Tags:     []string{"tag1", "tag2"},
				Remotes:  []*chshare.Remote{remote1, remote2},
				OS:       "unknown",
				OSArch:   "test-arch",
				OSFamily: "unknown",
				OSKernel: "unknown",
				Hostname: "unknown",
				IPv4:     nil,
				IPv6:     nil,
			},
		}, {
			Name: "uname error",
			SystemInfo: &mockSystemInfo{
				ReturnHostname:   "test-hostname",
				ReturnUnameError: errors.New("test error"),
				ReturnHostInfo: &host.InfoStat{
					OS:              "test-os",
					Platform:        "test-platform",
					PlatformVersion: "123",
					PlatformFamily:  "test-family",
				},
				ReturnInterfaceAddrs: interfaceAddrs,
				ReturnGoArch:         "test-arch",
			},
			ExpectedConnectionRequest: &chshare.ConnectionRequest{
				Version:  "0.0.0-src",
				ID:       "test-client-id",
				Name:     "test-name",
				Tags:     []string{"tag1", "tag2"},
				Remotes:  []*chshare.Remote{remote1, remote2},
				OS:       "unknown",
				OSArch:   "test-arch",
				OSFamily: "test-family",
				OSKernel: "test-os",
				Hostname: "test-hostname",
				IPv4:     []string{"192.0.2.1", "192.0.2.2"},
				IPv6:     []string{"2001:db8::1", "2001:db8::2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			client := NewClient(config)
			client.systemInfo = tc.SystemInfo

			connReq := client.connectionRequest(context.Background())

			assert.Equal(t, tc.ExpectedConnectionRequest, connReq)
		})
	}
}
