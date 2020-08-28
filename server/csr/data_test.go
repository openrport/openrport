package csr

import (
	"fmt"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

// nowMockF is used to override csr.now
var nowMockF = func() time.Time {
	return s2DisconnectedTime.Add(5 * time.Minute)
}

var (
	nowMock = nowMockF()
	// s2DisconnectedTime is "Random Rport Client 2" disconnected time
	s2DisconnectedTime, _ = time.Parse(time.RFC3339, "2020-08-19T13:04:23+03:00")
)

var s1JSON = `{
    "id": "aa1210c7-1899-491e-8e71-564cacaf1df8",
    "name": "Random Rport Client 1",
    "os": "Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
    "hostname": "alpine-3-10-tk-01",
    "ipv4": [
      "192.168.122.111"
    ],
    "ipv6": [
      "fe80::b84f:aff:fe59:a0b1"
    ],
    "tags": [
      "Linux",
      "Datacenter 1"
    ],
    "version": "0.1.12",
    "address": "88.198.189.161:50078",
    "tunnels": [
      {
        "lhost": "0.0.0.0",
        "lport": "2222",
        "rhost": "0.0.0.0",
        "rport": "22",
        "id": "1"
      },
      {
        "lhost": "0.0.0.0",
        "lport": "4000",
        "rhost": "0.0.0.0",
        "rport": "80",
        "id": "2"
      }
	]
  }`

var s1 = &ClientSession{
	ID:       "aa1210c7-1899-491e-8e71-564cacaf1df8",
	Name:     "Random Rport Client 1",
	OS:       "Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
	Hostname: "alpine-3-10-tk-01",
	IPv4:     []string{"192.168.122.111"},
	IPv6:     []string{"fe80::b84f:aff:fe59:a0b1"},
	Tags:     []string{"Linux", "Datacenter 1"},
	Version:  "0.1.12",
	Address:  "88.198.189.161:50078",
	Tunnels: []*Tunnel{
		{
			ID: "1",
			Remote: chshare.Remote{
				LocalHost:  "0.0.0.0",
				LocalPort:  "2222",
				RemoteHost: "0.0.0.0",
				RemotePort: "22",
			},
		},
		{
			ID: "2",
			Remote: chshare.Remote{
				LocalHost:  "0.0.0.0",
				LocalPort:  "4000",
				RemoteHost: "0.0.0.0",
				RemotePort: "80",
			},
		},
	},
	Disconnected: &nowMock,
}

var s2JSON = `{
    "id": "2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
    "name": "Random Rport Client 2",
    "os": "Linux alpine-3-10-tk-02 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
    "hostname": "alpine-3-10-tk-02",
    "ipv4": [
      "192.168.122.112"
    ],
    "ipv6": [
      "fe80::b84f:aff:fe59:a0b2"
    ],
    "tags": [
      "Linux",
      "Datacenter 2"
    ],
    "version": "0.1.12",
    "address": "88.198.189.162:50078",
    "tunnels": [
      {
        "lhost": "0.0.0.0",
        "lport": "2222",
        "rhost": "0.0.0.0",
        "rport": "22",
        "id": "1"
      }
	],
    "disconnected": "2020-08-19T13:04:23+03:00"
  }`

var s2 = &ClientSession{
	ID:       "2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
	Name:     "Random Rport Client 2",
	OS:       "Linux alpine-3-10-tk-02 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
	Hostname: "alpine-3-10-tk-02",
	IPv4:     []string{"192.168.122.112"},
	IPv6:     []string{"fe80::b84f:aff:fe59:a0b2"},
	Tags:     []string{"Linux", "Datacenter 2"},
	Version:  "0.1.12",
	Address:  "88.198.189.162:50078",
	Tunnels: []*Tunnel{
		{
			ID: "1",
			Remote: chshare.Remote{
				LocalHost:  "0.0.0.0",
				LocalPort:  "2222",
				RemoteHost: "0.0.0.0",
				RemotePort: "22",
			},
		},
	},
	Disconnected: &s2DisconnectedTime,
}

var s3JSON = `{
    "id": "c1d3c6811e1282c675495c0b3149dfa3201883188c42727a318d4a0742564c96",
    "name": "Random Rport Client 3",
    "os": "Linux alpine-3-10-tk-03 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
    "hostname": "alpine-3-10-tk-03",
    "ipv4": [
      "192.168.122.113"
    ],
    "ipv6": [
      "fe80::b84f:aff:fe59:a0b3"
    ],
    "tags": [
      "Linux",
      "Datacenter 3"
    ],
    "version": "0.1.12",
    "address": "88.198.189.163:50078",
    "tunnels": [],
    "disconnected": "2020-08-19T10:04:23+03:00"
  }`

var emptyFile = ``
var jsonEmptyArray = `[]`
var jsonOneEach = fmt.Sprintf("[%s,%s,%s]", s1JSON, s2JSON, s3JSON)
var jsonCorruptedWithOneClient = fmt.Sprintf("[%s,%s", s1JSON, `
 {
   "id": "2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
   "name": "Random Rport Client 2",
   "os": "Linux alpine-3-10-tk-02 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
   "host`)
