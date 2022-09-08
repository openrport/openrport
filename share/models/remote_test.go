package models

import (
	"fmt"
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
			WantRemoteHost: LocalHost,
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
			WantRemoteHost: LocalHost,
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
			Input:          "3000/tcp",
			WantProtocol:   ProtocolTCP,
			WantRemoteHost: LocalHost,
			WantRemotePort: "3000",
		},
		{
			Input:          "3000/udp",
			WantProtocol:   ProtocolUDP,
			WantRemoteHost: LocalHost,
			WantRemotePort: "3000",
		},
		{
			Input:          "3000/tcp+udp",
			WantProtocol:   ProtocolTCPUDP,
			WantRemoteHost: LocalHost,
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
			WantRemoteHost: LocalHost,
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

func TestIsProtocol(t *testing.T) {
	testCases := []struct {
		Protocol      string
		OtherProtocol string
		Expected      bool
	}{
		{
			Protocol:      ProtocolTCP,
			OtherProtocol: ProtocolTCP,
			Expected:      true,
		},
		{
			Protocol:      ProtocolUDP,
			OtherProtocol: ProtocolUDP,
			Expected:      true,
		},
		{
			Protocol:      ProtocolTCP,
			OtherProtocol: ProtocolUDP,
			Expected:      false,
		},
		{
			Protocol:      ProtocolTCPUDP,
			OtherProtocol: ProtocolTCP,
			Expected:      true,
		},
		{
			Protocol:      ProtocolTCP,
			OtherProtocol: ProtocolTCPUDP,
			Expected:      true,
		},
		{
			Protocol:      ProtocolTCPUDP,
			OtherProtocol: ProtocolUDP,
			Expected:      true,
		},
		{
			Protocol:      ProtocolUDP,
			OtherProtocol: ProtocolTCPUDP,
			Expected:      true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%s/%s", tc.Protocol, tc.OtherProtocol), func(t *testing.T) {
			t.Parallel()

			remote := &Remote{Protocol: tc.Protocol}

			result := remote.IsProtocol(tc.OtherProtocol)

			assert.Equal(t, tc.Expected, result)
		})
	}
}
