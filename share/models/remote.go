package models

import (
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// short-hand conversions
//   3000 ->
//     local  127.0.0.1:3000
//     remote 127.0.0.1:3000
//   foobar.com:3000 ->
//     local  127.0.0.1:3000
//     remote foobar.com:3000
//   3000:google.com:80 ->
//     local  127.0.0.1:3000
//     remote google.com:80
//   192.168.0.1:3000:google.com:80 ->
//     local  192.168.0.1:3000
//     remote google.com:80
//   .../udp ->  udp protocol

const (
	ZeroHost    = "0.0.0.0"
	ProtocolTCP = "tcp"
	ProtocolUDP = "udp"
)

var protocolRe = regexp.MustCompile(`(.*)\/(tcp|udp)$`)

// TODO(m-terel): Remote should be only used for parsing command args and URL query params. Current Remote is kind of a Tunnel model. Refactor to use separate models for representation and business logic.
type Remote struct {
	Protocol           string        `json:"protocol"`
	LocalHost          string        `json:"lhost"`
	LocalPort          string        `json:"lport"`
	RemoteHost         string        `json:"rhost"`
	RemotePort         string        `json:"rport"`
	LocalPortRandom    bool          `json:"lport_random"`
	Scheme             *string       `json:"scheme"`
	ACL                *string       `json:"acl"` // string representation of Tunnel.TunnelACL field
	IdleTimeoutMinutes int           `json:"idle_timeout_minutes"`
	AutoClose          time.Duration `json:"auto_close"`
	HTTPProxy          bool          `json:"http_proxy"`
	HostHeader         string        `json:"host_header"`
}

func DecodeRemote(s string) (*Remote, error) {
	protocol := ProtocolTCP
	matches := protocolRe.FindStringSubmatch(s)
	if len(matches) >= 3 {
		s = matches[1]
		protocol = matches[2]
	}

	parts := strings.Split(s, ":")
	if len(parts) <= 0 || len(parts) >= 5 {
		return nil, errors.New("Invalid remote")
	}

	r := &Remote{
		Protocol: protocol,
	}
	for i := len(parts) - 1; i >= 0; i-- {
		p := parts[i]
		if isPort(p) {
			if r.RemotePort == "" {
				r.RemotePort = p
			} else {
				r.LocalPort = p
			}
			continue
		}
		if r.RemotePort == "" && r.LocalPort == "" {
			return nil, errors.New("Missing ports")
		}
		if !isHost(p) {
			return nil, errors.New("Invalid host")
		}
		if r.RemoteHost == "" {
			r.RemoteHost = p
		} else {
			r.LocalHost = p
		}
	}
	if r.LocalHost == "" && r.LocalPort != "" {
		r.LocalHost = ZeroHost
	}
	if r.RemoteHost == "" {
		r.RemoteHost = ZeroHost
	}
	return r, nil
}

var isPortRegExp = regexp.MustCompile(`^\d+$`)

func isPort(s string) bool {
	return isPortRegExp.MatchString(s)
}

func isHost(s string) bool {
	_, err := url.Parse(s)
	return err == nil
}

//implement Stringer
func (r Remote) String() string {
	s := r.LocalHost + ":" + r.LocalPort + ":" + r.Remote()
	if r.Protocol != ProtocolTCP {
		s += "/" + r.Protocol
	}
	if r.ACL != nil {
		s += "(acl:" + *r.ACL + ")"
	}
	return s
}

func (r *Remote) Remote() string {
	return net.JoinHostPort(r.RemoteHost, r.RemotePort)
}

func (r *Remote) Local() string {
	return net.JoinHostPort(r.LocalHost, r.LocalPort)
}

func (r *Remote) Equals(other *Remote) bool {
	return r.String() == other.String()
}

func (r *Remote) EqualACL(acl *string) bool {
	if r.ACL != nil && acl != nil {
		return *r.ACL == *acl
	}
	if r.ACL == nil && acl == nil {
		return true
	}
	return false
}

func (r *Remote) IsLocalSpecified() bool {
	return r.LocalHost != "" && r.LocalPort != ""
}
