package chclient

import (
	"context"
	"net"

	"github.com/shirou/gopsutil/host"
)

type mockSystemInfo struct {
	ReturnHostname            string
	ReturnHostnameError       error
	ReturnHostInfo            *host.InfoStat
	ReturnHostInfoError       error
	ReturnUname               string
	ReturnUnameError          error
	ReturnInterfaceAddrs      []net.Addr
	ReturnInterfaceAddrsError error
	ReturnGoArch              string
}

func (s *mockSystemInfo) Hostname() (string, error) {
	return s.ReturnHostname, s.ReturnHostnameError
}

func (s *mockSystemInfo) HostInfo(ctx context.Context) (*host.InfoStat, error) {
	return s.ReturnHostInfo, s.ReturnHostInfoError
}

func (s *mockSystemInfo) Uname(ctx context.Context) (string, error) {
	return s.ReturnUname, s.ReturnUnameError
}

func (s *mockSystemInfo) InterfaceAddrs() ([]net.Addr, error) {
	return s.ReturnInterfaceAddrs, s.ReturnInterfaceAddrsError
}

func (s *mockSystemInfo) GoArch() string {
	return s.ReturnGoArch
}
