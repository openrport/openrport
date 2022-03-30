package chclient

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAllowed(t *testing.T) {
	lookupIP = func(host string) ([]net.IP, error) {
		if host == "example.com" {
			return []net.IP{
				net.ParseIP("192.0.2.2"),
				net.ParseIP("192.0.2.1"),
			}, nil
		}
		return []net.IP{
			net.ParseIP(host),
		}, nil
	}
	testCases := []struct {
		Name          string
		Remote        string
		TunnelAllowed []string
		Expected      bool
		ExpectedError string
	}{
		{
			Name:          "no config allowed",
			Remote:        "192.0.2.1:3000",
			TunnelAllowed: nil,
			Expected:      true,
		},
		{
			Name:          "ip allowed",
			Remote:        "192.0.2.1:3000",
			TunnelAllowed: []string{"192.0.2.1"},
			Expected:      true,
		},
		{
			Name:          "ip not allowed",
			Remote:        "192.0.2.1:3000",
			TunnelAllowed: []string{"192.0.2.2"},
			Expected:      false,
		},
		{
			Name:          "ip allowed last",
			Remote:        "192.0.2.1:3000",
			TunnelAllowed: []string{"3003", "192.0.3.0/24", "192.0.2.1"},
			Expected:      true,
		},
		{
			Name:          "hostname both ips allowed",
			Remote:        "example.com:3000",
			TunnelAllowed: []string{"192.0.2.1", "192.0.2.2"},
			Expected:      true,
		},
		{
			Name:          "hostname one ip allowed",
			Remote:        "example.com:3000",
			TunnelAllowed: []string{"192.0.2.1"},
			Expected:      false,
		},
		{
			Name:          "hostname no ips allowed",
			Remote:        "example.com:3000",
			TunnelAllowed: []string{"192.0.2.3"},
			Expected:      false,
		},
		{
			Name:          "ip range allowed",
			Remote:        "192.0.2.1:3000",
			TunnelAllowed: []string{"192.0.2.0/24"},
			Expected:      true,
		},
		{
			Name:          "ip range not allowed",
			Remote:        "192.0.3.1:3000",
			TunnelAllowed: []string{"192.0.2.0/24"},
			Expected:      false,
		},
		{
			Name:          "port allowed",
			Remote:        "192.0.2.1:3000",
			TunnelAllowed: []string{":3000"},
			Expected:      true,
		},
		{
			Name:          "port not allowed",
			Remote:        "192.0.2.1:3001",
			TunnelAllowed: []string{":3000"},
			Expected:      false,
		},
		{
			Name:          "ip and port allowed",
			Remote:        "192.0.2.1:3000",
			TunnelAllowed: []string{"192.0.2.1:3000"},
			Expected:      true,
		},
		{
			Name:          "ip and port ip not allowed",
			Remote:        "192.0.2.2:3000",
			TunnelAllowed: []string{"192.0.2.1:3000"},
			Expected:      false,
		},
		{
			Name:          "ip and port port not allowed",
			Remote:        "192.0.2.1:3001",
			TunnelAllowed: []string{"192.0.2.1:3000"},
			Expected:      false,
		},
		{
			Name:          "ip range and port allowed",
			Remote:        "192.0.2.1:3000",
			TunnelAllowed: []string{"192.0.2.0/24:3000"},
			Expected:      true,
		},
		{
			Name:          "ip range and port ip not allowed",
			Remote:        "192.0.3.1:3000",
			TunnelAllowed: []string{"192.0.2.0/24:3000"},
			Expected:      false,
		},
		{
			Name:          "ip range and port port not allowed",
			Remote:        "192.0.2.1:3001",
			TunnelAllowed: []string{"192.0.2.0/24:3000"},
			Expected:      false,
		},
		{
			Name:          "invalid remote",
			Remote:        "192.0.2.1",
			TunnelAllowed: []string{"192.0.2.0/24"},
			ExpectedError: "address 192.0.2.1: missing port in address",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			result, err := TunnelIsAllowed(tc.TunnelAllowed, tc.Remote)
			if tc.ExpectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.ExpectedError)
			}

			assert.Equal(t, tc.Expected, result)
		})
	}
}

func TestParseTunnelAllowed(t *testing.T) {
	testCases := []struct {
		Input         string
		ExpectedNet   *net.IPNet
		ExpectedPort  string
		ExpectedError string
	}{
		{
			Input:        "3000",
			ExpectedPort: "3000",
		},
		{
			Input:        ":3000",
			ExpectedPort: "3000",
		},
		{
			Input: "192.0.2.1",
			ExpectedNet: &net.IPNet{
				IP:   net.ParseIP("192.0.2.1"),
				Mask: net.CIDRMask(32, 32),
			},
		},
		{
			Input: "192.0.2.0/24",
			ExpectedNet: &net.IPNet{
				IP:   net.ParseIP("192.0.2.0"),
				Mask: net.CIDRMask(24, 32),
			},
		},
		{
			Input: "192.0.2.1:3000",
			ExpectedNet: &net.IPNet{
				IP:   net.ParseIP("192.0.2.1"),
				Mask: net.CIDRMask(32, 32),
			},
			ExpectedPort: "3000",
		},
		{
			Input: "192.0.2.0/24:3000",
			ExpectedNet: &net.IPNet{
				IP:   net.ParseIP("192.0.2.0"),
				Mask: net.CIDRMask(24, 32),
			},
			ExpectedPort: "3000",
		},
		{
			Input:         "",
			ExpectedError: `empty value not allowed: ""`,
		},
		{
			Input:         ":",
			ExpectedError: `empty value not allowed: ":"`,
		},
		{
			Input:         "abc",
			ExpectedError: `invalid port: "abc"`,
		},
		{
			Input:         "abc:3000",
			ExpectedError: `invalid ip range: "abc"`,
		},
		{
			Input:         "192.0.2.1:abc",
			ExpectedError: `invalid port: "abc"`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Input, func(t *testing.T) {
			t.Parallel()

			ipnet, port, err := ParseTunnelAllowed(tc.Input)
			if tc.ExpectedError != "" {
				assert.EqualError(t, err, tc.ExpectedError)
			} else {
				require.NoError(t, err)
				if tc.ExpectedNet != nil {
					assert.True(t, tc.ExpectedNet.IP.Equal(ipnet.IP))
					assert.Equal(t, tc.ExpectedNet.Mask, ipnet.Mask)
				} else {
					assert.Nil(t, ipnet)
				}
				assert.Equal(t, tc.ExpectedPort, port)
			}
		})
	}
}
