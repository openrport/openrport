package clienttunnel_test

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openrport/openrport/server/clients/clienttunnel"
)

func TestParseTunnelACL(t *testing.T) {
	testCases := []struct {
		Name          string
		Input         string
		Expected      []net.IPNet
		ExpectedError error
	}{
		{
			Name:     "empty",
			Input:    "",
			Expected: nil,
		},
		{
			Name:  "single ip",
			Input: "192.0.2.1",
			Expected: []net.IPNet{
				{IP: net.IPv4(192, 0, 2, 1), Mask: net.CIDRMask(32, 32)},
			},
		},
		{
			Name:  "ip range",
			Input: "192.0.2.0/24",
			Expected: []net.IPNet{
				{IP: net.IPv4(192, 0, 2, 0), Mask: net.CIDRMask(24, 32)},
			},
		},
		{
			Name:  "multiple entries",
			Input: "192.0.2.1,192.0.2.2/31",
			Expected: []net.IPNet{
				{IP: net.IPv4(192, 0, 2, 1), Mask: net.CIDRMask(32, 32)},
				{IP: net.IPv4(192, 0, 2, 2), Mask: net.CIDRMask(31, 32)},
			},
		},
		{
			Name:          "zero ip",
			Input:         "0.0.0.0",
			ExpectedError: errors.New("0.0.0.0 would allow access to everyone. If that's what you want, do not set the ACL"),
		},
		{
			Name:  "ipv6",
			Input: "2001:db8:3333:4444:5555:6666:7777:8888",
			Expected: []net.IPNet{
				{IP: net.ParseIP("2001:db8:3333:4444:5555:6666:7777:8888"), Mask: net.CIDRMask(128, 128)},
			},
		},
		{
			Name:  "ipv6 range",
			Input: "2001:db8:3333:4444::/64",
			Expected: []net.IPNet{
				{IP: net.ParseIP("2001:db8:3333:4444::"), Mask: net.CIDRMask(64, 128)},
			},
		},
		{
			Name:  "mixed entries",
			Input: "::1,127.0.0.0/24",
			Expected: []net.IPNet{
				{IP: net.ParseIP("::1"), Mask: net.CIDRMask(128, 128)},
				{IP: net.ParseIP("127.0.0.0"), Mask: net.CIDRMask(24, 32)},
			},
		},
		{
			Name:          "zero ipv6",
			Input:         "::",
			ExpectedError: errors.New(":: would allow access to everyone. If that's what you want, do not set the ACL"),
		},
		{
			Name:          "invalid ip",
			Input:         "invalid.ip",
			ExpectedError: errors.New("invalid IP addr: invalid.ip"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			result, err := clienttunnel.ParseTunnelACL(tc.Input)

			assert.Equal(t, tc.ExpectedError, err)
			if tc.Expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, len(tc.Expected), len(result.AllowedIPs))
				for i := range tc.Expected {
					assert.True(t, tc.Expected[i].IP.Equal(result.AllowedIPs[i].IP))
					assert.Equal(t, tc.Expected[i].Mask, result.AllowedIPs[i].Mask)
				}
			}
		})
	}
}
