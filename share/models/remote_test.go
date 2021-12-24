package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeRemote(t *testing.T) {
	testCases := []struct {
		Input          string
		WantProtocol   string
		WantLocalHost  string
		WantLocalPort  string
		WantRemoteHost string
		WantRemotePort string
	}{
		{
			Input:          "3000",
			WantProtocol:   ProtocolTCP,
			WantRemoteHost: ZeroHost,
			WantRemotePort: "3000",
		},
		{
			Input:          "foobar.com:3000",
			WantProtocol:   ProtocolTCP,
			WantRemoteHost: "foobar.com",
			WantRemotePort: "3000",
		},
		{
			Input:          "3000:google.com:80",
			WantProtocol:   ProtocolTCP,
			WantLocalHost:  ZeroHost,
			WantLocalPort:  "3000",
			WantRemoteHost: "google.com",
			WantRemotePort: "80",
		},
		{
			Input:          "3000:80",
			WantProtocol:   ProtocolTCP,
			WantLocalHost:  ZeroHost,
			WantLocalPort:  "3000",
			WantRemoteHost: ZeroHost,
			WantRemotePort: "80",
		},
		{
			Input:          "192.168.0.1:3000:google.com:80",
			WantProtocol:   ProtocolTCP,
			WantLocalHost:  "192.168.0.1",
			WantLocalPort:  "3000",
			WantRemoteHost: "google.com",
			WantRemotePort: "80",
		},
		{
			Input:          "3000/udp",
			WantProtocol:   ProtocolUDP,
			WantRemoteHost: ZeroHost,
			WantRemotePort: "3000",
		},
		{
			Input:          "foobar.com:3000/udp",
			WantProtocol:   ProtocolUDP,
			WantRemoteHost: "foobar.com",
			WantRemotePort: "3000",
		},
		{
			Input:          "3000:google.com:80/udp",
			WantProtocol:   ProtocolUDP,
			WantLocalHost:  ZeroHost,
			WantLocalPort:  "3000",
			WantRemoteHost: "google.com",
			WantRemotePort: "80",
		},
		{
			Input:          "3000:80/udp",
			WantProtocol:   ProtocolUDP,
			WantLocalHost:  ZeroHost,
			WantLocalPort:  "3000",
			WantRemoteHost: ZeroHost,
			WantRemotePort: "80",
		},
		{
			Input:          "192.168.0.1:3000:google.com:80/udp",
			WantProtocol:   ProtocolUDP,
			WantLocalHost:  "192.168.0.1",
			WantLocalPort:  "3000",
			WantRemoteHost: "google.com",
			WantRemotePort: "80",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Input, func(t *testing.T) {
			t.Parallel()

			remote, err := DecodeRemote(tc.Input)
			require.NoError(t, err)
			assert.Equal(t, tc.WantProtocol, remote.Protocol)
			assert.Equal(t, tc.WantLocalHost, remote.LocalHost)
			assert.Equal(t, tc.WantLocalPort, remote.LocalPort)
			assert.Equal(t, tc.WantRemoteHost, remote.RemoteHost)
			assert.Equal(t, tc.WantRemotePort, remote.RemotePort)
		})
	}
}
