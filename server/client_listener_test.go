package chserver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share/models"
)

func TestGetTunnelsToReestablish(t *testing.T) {
	var randomPorts = []string{"5001", "5002", "5003", "5004", "5005", "5006", "5007", "5008", "5009"}
	testCases := []struct {
		descr string // Test Case Description

		oldStr []string
		oldACL []string
		newStr []string
		newACL []string

		wantResStr []string
	}{
		{
			descr:      "both empty",
			oldStr:     nil,
			newStr:     nil,
			wantResStr: []string{},
		},
		{
			descr: "no new tunnels",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			newStr: []string{},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
				"0.0.0.0:3000:site.com:80",
				"::foobar.com:3000",
				"::0.0.0.0:3000",
			},
		},
		{
			descr:  "no old tunnels",
			oldStr: []string{},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			wantResStr: nil,
		},
		{
			descr: "same tunnels specified in 4 possible forms",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			wantResStr: nil,
		},
		{
			descr: "old tunnels include all new",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
				"192.168.0.1:3001:google.com:80",
				"3001:site.com:80",
				"foobar.com:3001",
				"3001",
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			wantResStr: []string{
				"192.168.0.1:3001:google.com:80",
				"0.0.0.0:3001:site.com:80",
				"::foobar.com:3001",
				"::0.0.0.0:3001",
			},
		},
		{
			descr: "old tunnels were with random ports, but new has the same random ports",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"3000",          //contains randomPorts[2]
				"foobar.com:22", //contains randomPorts[3]
				"3000",          //contains randomPorts[4]
			},
			newStr: []string{
				"0.0.0.0:" + randomPorts[1] + ":foobar.com:22",
				"0.0.0.0:" + randomPorts[2] + ":0.0.0.0:3000",
				"0.0.0.0:" + randomPorts[3] + ":foobar.com:22",
				"0.0.0.0:" + randomPorts[4] + ":0.0.0.0:3000",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
		},
		{
			descr: "old tunnels were with random ports, but new has 2 the same random ports and 2 random",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"3000",          //contains randomPorts[2]
				"foobar.com:22", //contains randomPorts[3]
				"3000",          //contains randomPorts[4]
			},
			newStr: []string{
				"0.0.0.0:" + randomPorts[1] + ":foobar.com:22",
				"0.0.0.0:" + randomPorts[2] + ":0.0.0.0:3000",
				"foobar.com:22",
				"3000",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
		},
		{
			descr: "old tunnels were with random ports, but new has the different random port",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"foobar.com:22", //contains randomPorts[2]
			},
			newStr: []string{
				"0.0.0.0:" + randomPorts[2] + ":foobar.com:22",
				"0.0.0.0:" + randomPorts[3] + ":foobar.com:22",
			},
			wantResStr: nil,
		},
		{
			descr: "old tunnels are with random port 1 and 2, new tunnels are with random port and a port that equals to random port 1",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"foobar.com:22", //contains randomPorts[2]
			},
			newStr: []string{
				"foobar.com:22",
				"0.0.0.0:" + randomPorts[1] + ":foobar.com:22",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
		},
		{
			descr: "old tunnels are with random port 1 and 2, new tunnels are with a port that equals to random port 1 and a random port",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"foobar.com:22", //contains randomPorts[1]
				"foobar.com:22", //contains randomPorts[2]
			},
			// different order to a previous test case
			newStr: []string{
				"0.0.0.0:" + randomPorts[1] + ":foobar.com:22",
				"foobar.com:22",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
		},
		{
			descr: "old tunnels include all new, multiple similar with random port",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"192.168.0.1:3000:google.com:8080",
				"3000:site.com:80",
				"foobar.com:3000", //contains randomPorts[4]
				"foobar.com:3000", //contains randomPorts[5]
				"foobar.com:3000", //contains randomPorts[6]
				"3000",            //contains randomPorts[7]
				"3000",            //contains randomPorts[8]
				"3000",            //contains randomPorts[9]
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"0.0.0.0:" + randomPorts[4] + ":foobar.com:3000",
				"foobar.com:3000",
				"foobar.com:3000",
				"3000",
				"3000",
				"0.0.0.0:" + randomPorts[7] + ":0.0.0.0:3000",
			},
			wantResStr: []string{
				"192.168.0.1:3000:google.com:8080",
			},
		},
		{
			descr: "new tunnels include all old",
			oldStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
				"3000:site.com:80",
				"foobar.com:3000",
				"3000",
				"192.168.0.1:3001:google.com:80",
				"3001:site.com:80",
				"foobar.com:3001",
				"3001",
			},
			wantResStr: nil,
		},
		{
			descr: "new tunnel specified in form '<local-host>:<local-port>:<remote-host>:<remote-port>' is not among old",
			oldStr: []string{
				"192.168.0.2:3000:google.com:80",
				"192.168.0.1:3001:google.com:80",
				"192.168.0.1:3000:google.com.ua:80",
				"192.168.0.1:3000:google.com:8080",
				"3000:google.com:80",
				"google.com:80",
				"80",
			},
			newStr: []string{
				"192.168.0.1:3000:google.com:80",
			},
			wantResStr: nil,
		},
		{
			descr: "new tunnel specified in form '<local-port>:<remote-host>:<remote-port>' is not among old",
			oldStr: []string{
				"192.168.0.1:3000:site.com:80",
				"3001:site.com:80",
				"3000:site-2.com:80",
				"3000:site.com:22",
				"site.com:80",
				"80",
			},
			newStr: []string{
				"3000:site.com:80",
			},
			wantResStr: nil,
		},
		{
			descr: "new tunnel specified in form '<remote-host>:<remote-port>' is not among old",
			oldStr: []string{
				"192.168.0.1:3000:foobar.com:3000",
				"0.0.0.0:3001:foobar.com:3000",
				"3000:foobar.com:3000",
				"foobar.com:3001",
				"foobar-2.com:3000",
				"3000",
			},
			newStr: []string{
				"foobar.com:3000",
			},
			wantResStr: nil,
		},
		{
			descr: "new tunnel specified in form '<remote-port>' is not among old",
			oldStr: []string{
				"192.168.0.1:3000:foobar.com:3000",
				"0.0.0.0:3000:0.0.0.0:3000",
				"3000:0.0.0.0:3000",
				"3000:foobar.com:3000",
				"foobar.com:3000",
				"3001",
			},
			newStr: []string{
				"3000",
			},
			wantResStr: nil,
		},
		{
			descr: "same old and new tunnel but different ACLs",
			oldStr: []string{
				"5432:0.0.0.0:22",
			},
			oldACL: []string{
				"95.67.52.213",
			},
			newStr: []string{
				"5432:0.0.0.0:22",
			},
			newACL: []string{
				"95.67.52.214",
			},
			wantResStr: nil,
		},
		{
			descr: "same old and new tunnel without local but different ACLs",
			oldStr: []string{
				"22",
			},
			oldACL: []string{
				"95.67.52.213",
			},
			newStr: []string{
				"22",
			},
			newACL: []string{
				"95.67.52.214",
			},
			wantResStr: nil,
		},
		{
			descr: "old tunnels have 2 similar tunnels but different ACLs, new tunnels contains one of them",
			oldStr: []string{
				"2222:0.0.0.0:22",
				"3333:0.0.0.0:22",
			},
			oldACL: []string{
				"95.67.52.213",
				"95.67.52.214",
			},
			newStr: []string{
				"2222:0.0.0.0:22",
			},
			newACL: []string{
				"95.67.52.213",
			},
			wantResStr: []string{
				"0.0.0.0:3333:0.0.0.0:22(acl:95.67.52.214)",
			},
		},
		{
			descr: "old and new tunnels have 2 same tunnels without local but different ACLs",
			oldStr: []string{
				"22",
				"22",
			},
			oldACL: []string{
				"95.67.52.213",
				"95.67.52.214",
			},
			newStr: []string{
				"22",
				"22",
			},
			newACL: []string{
				"95.67.52.213",
				"95.67.52.214",
			},
			wantResStr: nil,
		},
		{
			descr: "old tunnels have 3 same tunnels without local but different ACLs, new tunnels have 2 of them",
			oldStr: []string{
				"22",
				"22",
				"22",
			},
			oldACL: []string{
				"95.67.52.213",
				"95.67.52.214",
				"95.67.52.215",
			},
			newStr: []string{
				"22",
				"22",
			},
			newACL: []string{
				"95.67.52.213",
				"95.67.52.214",
			},
			wantResStr: []string{
				"::0.0.0.0:22(acl:95.67.52.215)",
			},
		},
	}
	for _, tc := range testCases {
		msg := fmt.Sprintf("test case: %q", tc.descr)

		// given
		var old, new []*chshare.Remote
		for i, v := range tc.oldStr {
			r, err := chshare.DecodeRemote(v)
			require.NoErrorf(t, err, msg)
			// mimic real behavior
			if !r.IsLocalSpecified() {
				r.LocalHost = "0.0.0.0"
				r.LocalPort = randomPorts[i]
				r.LocalPortRandom = true
			}
			if tc.oldACL != nil && tc.oldACL[i] != "" {
				r.ACL = &tc.oldACL[i]
			}
			old = append(old, r)
		}
		for i, v := range tc.newStr {
			r, err := chshare.DecodeRemote(v)
			require.NoErrorf(t, err, msg)
			if tc.newACL != nil && tc.newACL[i] != "" {
				r.ACL = &tc.newACL[i]
			}
			new = append(new, r)
		}

		// when
		gotRes := GetTunnelsToReestablish(old, new)

		var gotResStr []string
		for _, r := range gotRes {
			gotResStr = append(gotResStr, r.String())
		}

		// then
		assert.ElementsMatch(t, tc.wantResStr, gotResStr, msg)
	}
}
