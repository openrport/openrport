package chclient

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/share/test"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"golang.org/x/crypto/ssh"

	"github.com/shirou/gopsutil/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/client/system"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/clientconfig"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func TestCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "Bar", req.Header.Get("Foo"))
	}))
	// Close the server when test finishes
	defer server.Close()

	config := ClientConfigHolder{
		Config: &clientconfig.Config{
			Client: clientconfig.ClientConfig{
				Fingerprint: "",
				Auth:        "",
				Server:      server.URL,
				Remotes:     []string{"192.168.0.5:3000:google.com:80"},
				DataDir:     "somedir",
			},
			Connection: clientconfig.ConnectionConfig{
				KeepAlive:        time.Second,
				MaxRetryCount:    0,
				MaxRetryInterval: time.Second,
				HeadersRaw:       []string{"Foo: Bar"},
			},
			RemoteCommands: clientconfig.CommandsConfig{
				Order: allowDenyOrder,
			},
			RemoteScripts: clientconfig.ScriptsConfig{
				Enabled: false,
			},
		},
	}
	err := config.ParseAndValidate(true)
	require.NoError(t, err)

	fileAPI := test.NewFileAPIMock()
	c := NewClient(&config, fileAPI)
	require.NoError(t, err)

	err = c.Run()
	require.NoError(t, err)
}

func TestConnectionRequest(t *testing.T) {
	remote1 := &models.Remote{
		LocalHost:  "test-local",
		LocalPort:  "1234",
		RemoteHost: "test-remote",
		RemotePort: "2345",
	}
	remote2 := &models.Remote{
		LocalHost:  "test-local-2",
		LocalPort:  "2234",
		RemoteHost: "test-remote-2",
		RemotePort: "3345",
	}
	config := &ClientConfigHolder{
		Config: &clientconfig.Config{
			Client: clientconfig.ClientConfig{
				ID:      "test-client-id",
				Name:    "test-name",
				Tags:    []string{"tag1", "tag2"},
				Tunnels: []*models.Remote{remote1, remote2},
			},
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
		SystemInfo                system.SysInfo
		ExpectedConnectionRequest *chshare.ConnectionRequest
	}{
		{
			Name: "no errors",
			SystemInfo: &system.MockSystemInfo{
				ReturnUname:         "test-uname",
				ReturnHostname:      "test-hostname",
				ReturnHostnameError: nil,
				ReturnHostInfo: &host.InfoStat{
					OS:                   "test-os",
					PlatformFamily:       "test-family",
					Platform:             "UBUNTU",
					PlatformVersion:      "18.04",
					VirtualizationSystem: "dummy",
					VirtualizationRole:   "guest",
				},
				ReturnInterfaceAddrs: interfaceAddrs,
				ReturnGoArch:         "test-arch",
				ReturnCPUInfo: system.CPUInfo{
					CPUs: []cpu.InfoStat{
						{
							Family:    "fam1",
							Model:     "mod1",
							ModelName: "mod name 123",
							VendorID:  "GenuineIntel",
						},
					},
					NumCores: 4,
				},
				ReturnCPUPercent: 0.0,
				ReturnMemoryStat: &mem.VirtualMemoryStat{
					Total: 100000,
				},
				ReturnSystemTime: time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC),
			},
			ExpectedConnectionRequest: &chshare.ConnectionRequest{
				NumCPUs:                4,
				MemoryTotal:            100000,
				Version:                "0.0.0-src",
				ID:                     "test-client-id",
				Name:                   "test-name",
				OS:                     "test-uname",
				OSFullName:             "Ubuntu 18.04",
				OSVersion:              "18.04",
				OSVirtualizationSystem: "dummy",
				OSVirtualizationRole:   "guest",
				OSArch:                 "test-arch",
				OSFamily:               "test-family",
				OSKernel:               "test-os",
				Hostname:               "test-hostname",
				CPUFamily:              "fam1",
				CPUModel:               "mod1",
				CPUModelName:           "mod name 123",
				CPUVendor:              "GenuineIntel",
				Timezone:               "UTC (UTC+00:00)",
				IPv4:                   []string{"192.0.2.1", "192.0.2.2"},
				IPv6:                   []string{"2001:db8::1", "2001:db8::2"},
				Tags:                   []string{"tag1", "tag2"},
				Remotes:                []*models.Remote{remote1, remote2},
				ClientConfiguration:    config.Config,
			},
		}, {
			Name: "windows, no errors",
			SystemInfo: &system.MockSystemInfo{
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
				ReturnCPUInfo: system.CPUInfo{
					CPUs: []cpu.InfoStat{
						{
							Family:    "cpufam1",
							Model:     "cpumod1",
							ModelName: "cpumod_name1",
							VendorID:  "GenuineIntel",
						},
					},
					NumCores: 2,
				},
				ReturnSystemTime: time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC),
			},
			ExpectedConnectionRequest: &chshare.ConnectionRequest{
				Version:                "0.0.0-src",
				ID:                     "test-client-id",
				Name:                   "test-name",
				Tags:                   []string{"tag1", "tag2"},
				Remotes:                []*models.Remote{remote1, remote2},
				OS:                     "test-platform 123 test-family",
				OSArch:                 "test-arch",
				OSFamily:               "test-family",
				OSKernel:               "windows",
				OSVirtualizationSystem: "dummy",
				OSVirtualizationRole:   "guest",
				Hostname:               "test-hostname",
				OSFullName:             "Test-Platform 123",
				OSVersion:              "123",
				CPUFamily:              "cpufam1",
				CPUModel:               "cpumod1",
				CPUModelName:           "cpumod_name1",
				CPUVendor:              "GenuineIntel",
				Timezone:               "UTC (UTC+00:00)",
				NumCPUs:                2,
				IPv4:                   []string{"192.0.2.1", "192.0.2.2"},
				IPv6:                   []string{"2001:db8::1", "2001:db8::2"},
				ClientConfiguration:    config.Config,
			},
		}, {
			Name: "all errors",
			SystemInfo: &system.MockSystemInfo{
				ReturnHostnameError:       errors.New("test error"),
				ReturnUnameError:          errors.New("test error"),
				ReturnHostInfoError:       errors.New("test error"),
				ReturnInterfaceAddrsError: errors.New("test error"),
				ReturnGoArch:              "test-arch",
				ReturnCPUInfoError:        errors.New("test error"),
				ReturnMemoryError:         errors.New("test error"),
			},
			ExpectedConnectionRequest: &chshare.ConnectionRequest{
				Version:                "0.0.0-src",
				ID:                     "test-client-id",
				Name:                   "test-name",
				Tags:                   []string{"tag1", "tag2"},
				Remotes:                []*models.Remote{remote1, remote2},
				OS:                     system.UnknownValue,
				OSArch:                 "test-arch",
				OSFamily:               system.UnknownValue,
				OSKernel:               system.UnknownValue,
				Hostname:               system.UnknownValue,
				CPUFamily:              system.UnknownValue,
				CPUModel:               system.UnknownValue,
				CPUModelName:           system.UnknownValue,
				CPUVendor:              system.UnknownValue,
				OSFullName:             system.UnknownValue,
				OSVersion:              system.UnknownValue,
				OSVirtualizationSystem: "dummy",
				OSVirtualizationRole:   "guest",
				Timezone:               "UTC (UTC+00:00)",
				IPv4:                   nil,
				IPv6:                   nil,
				ClientConfiguration:    config.Config,
			},
		}, {
			Name: "uname error",
			SystemInfo: &system.MockSystemInfo{
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
				ReturnSystemTime:     time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC),
			},
			ExpectedConnectionRequest: &chshare.ConnectionRequest{
				Version:                "0.0.0-src",
				ID:                     "test-client-id",
				Name:                   "test-name",
				OSVersion:              "123",
				OSFullName:             "Test-Platform 123",
				Tags:                   []string{"tag1", "tag2"},
				Remotes:                []*models.Remote{remote1, remote2},
				OS:                     system.UnknownValue,
				OSArch:                 "test-arch",
				OSFamily:               "test-family",
				OSKernel:               "test-os",
				Hostname:               "test-hostname",
				Timezone:               "UTC (UTC+00:00)",
				CPUFamily:              system.UnknownValue,
				CPUModel:               system.UnknownValue,
				CPUModelName:           system.UnknownValue,
				CPUVendor:              system.UnknownValue,
				OSVirtualizationSystem: "dummy",
				OSVirtualizationRole:   "guest",
				IPv4:                   []string{"192.0.2.1", "192.0.2.2"},
				IPv6:                   []string{"2001:db8::1", "2001:db8::2"},
				ClientConfiguration:    config.Config,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			client := NewClient(config, test.NewFileAPIMock())

			client.systemInfo = tc.SystemInfo

			connReq, err := client.connectionRequest(context.Background())
			require.NoError(t, err)

			assert.Equal(t, tc.ExpectedConnectionRequest, connReq)
		})
	}
}

// mockServer receives client connections and keeps track whether the connection is established
type mockServer struct {
	upgrader  websocket.Upgrader
	sshConfig *ssh.ServerConfig

	mtx           sync.Mutex
	isUnavailable bool
	isConnected   bool
	sshConn       ssh.Conn
}

func newMockServer() (*mockServer, error) {
	key, err := chshare.GenerateKey("test")
	if err != nil {
		return nil, err
	}
	private, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	sshConfig := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	sshConfig.AddHostKey(private)

	return &mockServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		sshConfig: sshConfig,
	}, nil
}

func (m *mockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mtx.Lock()
	isUnavailable := m.isUnavailable
	m.mtx.Unlock()
	if isUnavailable {
		http.Error(w, "not available", http.StatusServiceUnavailable)
		return
	}

	wsConn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	conn := chshare.NewWebSocketConn(wsConn)
	sshConn, _, reqs, err := ssh.NewServerConn(conn, m.sshConfig)
	if err != nil {
		log.Println(err)
		return
	}
	m.mtx.Lock()
	m.sshConn = sshConn
	m.mtx.Unlock()

	req := <-reqs
	err = req.Reply(true, []byte("[]"))
	if err != nil {
		log.Println(err)
		return
	}
	m.mtx.Lock()
	m.isConnected = true
	m.mtx.Unlock()

	defer func() {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		m.isConnected = false
		m.sshConn = nil
	}()

	err = sshConn.Wait()
	if err != nil {
		log.Println(err)
		return
	}
}

func (m *mockServer) IsConnected() bool {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.isConnected
}

func (m *mockServer) WaitForStatus(isConnected bool) error {
	for i := 0; i < 500; i++ {
		if m.IsConnected() == isConnected {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for isConnected=%v", isConnected)
}

func (m *mockServer) SetAvailable(isAvailable bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.isUnavailable = !isAvailable
}

func (m *mockServer) CloseConnection() {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if m.sshConn != nil {
		m.sshConn.Close()
	}
}

func TestConnectionLoop(t *testing.T) {
	mainServer, err := newMockServer()
	require.NoError(t, err)
	tsMain := httptest.NewServer(mainServer)
	defer tsMain.Close()

	fallbackServer, err := newMockServer()
	require.NoError(t, err)
	tsFallback := httptest.NewServer(fallbackServer)
	defer tsFallback.Close()

	logOutput := logger.NewLogOutput("")
	err = logOutput.Start()
	require.NoError(t, err)

	config := ClientConfigHolder{
		Config: &clientconfig.Config{
			Client: clientconfig.ClientConfig{
				Server:                   tsMain.URL,
				FallbackServers:          []string{tsFallback.URL},
				ServerSwitchbackInterval: 100 * time.Millisecond,
				DataDir:                  "./",
			},
			RemoteCommands: clientconfig.CommandsConfig{
				Order: allowDenyOrder,
			},
			Logging: clientconfig.LogConfig{
				LogLevel:  logger.LogLevelDebug,
				LogOutput: logOutput,
			},
			Connection: clientconfig.ConnectionConfig{
				MaxRetryCount: -1,
			},
		},
	}
	err = config.ParseAndValidate(true)
	require.NoError(t, err)

	c := NewClient(&config, test.NewFileAPIMock())

	go c.connectionLoop(context.Background())

	// connects to main server successfully
	assert.NoError(t, mainServer.WaitForStatus(true))

	// retries connection to main server if it drops
	mainServer.CloseConnection()
	assert.NoError(t, mainServer.WaitForStatus(false))
	assert.NoError(t, mainServer.WaitForStatus(true))

	// connects to fallback if main server is down
	mainServer.SetAvailable(false)
	mainServer.CloseConnection()
	assert.NoError(t, mainServer.WaitForStatus(false))
	assert.NoError(t, fallbackServer.WaitForStatus(true))

	// stays connected to fallback while main server id down
	assert.NoError(t, mainServer.WaitForStatus(false))
	assert.NoError(t, fallbackServer.WaitForStatus(true))

	// connects back to main server when it becomes available
	mainServer.SetAvailable(true)
	assert.NoError(t, mainServer.WaitForStatus(true))
	assert.NoError(t, fallbackServer.WaitForStatus(false))
}
