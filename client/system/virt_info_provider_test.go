package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type virtInfoTestCase struct {
	rawData            string
	virtSystemToExpect string
	virtRoleToExpect   string
}

func TestGetVirtInfoFromPowershellServicesList(t *testing.T) {
	testCases := []virtInfoTestCase{
		{
			rawData: `
Stopped  PerfHost           Performance Counter DLL Host
Stopped  PhoneSvc           Phone Service
Stopped  PimIndexMainten... Contact Data_1400f45
Stopped  pla                Performance Logs & Alerts
Running  PlugPlay           Plug and Play
Running  PolicyAgent        IPsec Policy Agent
Running  Power              Power
Stopped  PrintNotify        Printer Extensions and Notifications
Running  ProfSvc            User Profile Service
Stopped  PhoneSvc           Phone Service
Stopped  PimIndexMainten... Contact Data_1400f45
Stopped  pla                Performance Logs & Alerts
Running  PlugPlay           Plug and Play
Running  PolicyAgent        IPsec Policy Agent
Running  Power              Power
Stopped  PrintNotify        Printer Extensions and Notifications
Running  ProfSvc            User Profile Service
Running  Pulse              Pulse by freeping.io
Stopped  QEMU Guest Agen... QEMU Guest Agent VSS Provider
Running  QEMU-GA            QEMU Guest Agent
Stopped  WiaRpc             Still Image Acquisition Events
`,
			virtSystemToExpect: VirtualSystemKVM,
			virtRoleToExpect:   VirtualSystemRoleGuest,
		},
		{
			rawData: `
Stopped  PerfHost           Performance Counter DLL Host
Stopped  PhoneSvc           Phone Service
Stopped  PimIndexMainten... Contact Data_1400f45
Stopped  pla                Performance Logs & Alerts
Running  PlugPlay           Plug and Play
Running  PolicyAgent        IPsec Policy Agent
Running  Power              Power
Stopped  PrintNotify        Printer Extensions and Notifications
Running  ProfSvc            User Profile Service
Running  VMTools        	VMWare Service
Stopped  PimIndexMainten... Contact Data_1400f45
Stopped  pla                Performance Logs & Alerts
Running  PlugPlay           Plug and Play
Running  PolicyAgent        IPsec Policy Agent
Running  Power              Power
Stopped  PrintNotify        Printer Extensions and Notifications
Running  ProfSvc            User Profile Service
Running  Pulse              Pulse by freeping.io
Stopped  WiaRpc             Still Image Acquisition Events
`,
			virtRoleToExpect:   VirtualSystemRoleGuest,
			virtSystemToExpect: VirtualSystemVMWare,
		},
		{
			rawData: `
Stopped  AppIDSvc           Application Identity
Stopped  Appinfo            Application Information
Stopped  AppMgmt            Application Management
Stopped  AppReadiness       App Readiness
Stopped  AppVClient         Microsoft App-V Client
Stopped  AppXSvc            AppX Deployment Service (AppXSVC)
Stopped  AudioEndpointBu... Windows Audio Endpoint Builder
Stopped  Audiosrv           Windows Audio
Stopped  AxInstSV           ActiveX Installer (AxInstSV)
Running  BFE                Base Filtering Engine
Stopped  BITS               Background Intelligent Transfer Ser...
Running  BrokerInfrastru... Background Tasks Infrastructure Ser...
Stopped  Browser            Computer Browser
Stopped  bthserv            Bluetooth Support Service
Running  CDPSvc             Connected Devices Platform Service
Running  CDPUserSvc_1400f45 CDPUserSvc_1400f45
Running  CertPropSvc        Certificate Propagation
Stopped  ClipSVC            Client License Service (ClipSVC)
Running  vmicheartbeat      Hyper-V Service
`,
			virtSystemToExpect: VirtualSystemHyperV,
			virtRoleToExpect:   VirtualSystemRoleGuest,
		},
		{
			rawData: `
Stopped  AppIDSvc           Application Identity
Stopped  Appinfo            Application Information
Stopped  AppMgmt            Application Management
Stopped  AppReadiness       App Readiness
Stopped  AppVClient         Microsoft App-V Client
Stopped  AppXSvc            AppX Deployment Service (AppXSVC)
Stopped  AudioEndpointBu... Windows Audio Endpoint Builder
Stopped  Audiosrv           Windows Audio
Stopped  AxInstSV           ActiveX Installer (AxInstSV)
Running  BFE                Base Filtering Engine
Stopped  BITS               Background Intelligent Transfer Ser...
Running  BrokerInfrastru... Background Tasks Infrastructure Ser...
Stopped  Browser            Computer Browser
Stopped  bthserv            Bluetooth Support Service
Running  CDPSvc             Connected Devices Platform Service
Running  CDPUserSvc_1400f45 CDPUserSvc_1400f45
Running  vmcompute        	Hyper-V Service
`,
			virtRoleToExpect:   VirtualSystemRoleHost,
			virtSystemToExpect: VirtualSystemHyperV,
		},
		{
			rawData: `
Stopped  AppIDSvc           Application Identity
Stopped  Appinfo            Application Information
Stopped  AppMgmt            Application Management
Stopped  AppReadiness       App Readiness
Stopped  AppVClient         Microsoft App-V Client
Stopped  AppXSvc            AppX Deployment Service (AppXSVC)
Stopped  AudioEndpointBu... Windows Audio Endpoint Builder
`,
			virtRoleToExpect:   UnknownValue,
			virtSystemToExpect: UnknownValue,
		},
	}

	for _, testCase := range testCases {
		virtSystemGiven, virtRoleGiven := getVirtInfoFromPowershellServicesList(testCase.rawData)

		assert.Equal(t, testCase.virtRoleToExpect, virtRoleGiven)
		assert.Equal(t, testCase.virtSystemToExpect, virtSystemGiven)
	}
}

func TestGetVirtInfoFromNixDevicesList(t *testing.T) {
	cases := []struct {
		rawData            string
		expectedVirtSystem string
		expectedVirtRole   string
	}{
		{
			rawData: `
0000	80861237	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
0008	80867000	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
0009	80867010	0	             1f0	             3f6	             170	             376	            e0a1	               0	               0	               8	               0	               8	               0	              10	               0	               0	ata_piix
000a	80867020	b	               0	               0	               0	               0	            e041	               0	               0	               0	               0	               0	               0	              20	               0	               0	uhci_hcd
000b	80867113	9	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	piix4_smbus
0010	12341111	0	        fd000008	               0	        fea50000	               0	               0	               0	           c0002	         1000000	               0	            1000	               0	               0	               0	           20000
0018	1af41002	a	            e061	               0	               0	               0	        fe40000c	               0	               0	              20	               0	               0	               0	            4000	               0	               0	virtio-pci
0028	1af41004	a	            e001	        fea51000	               0	               0	        fe40400c	               0	               0	              40	            1000	               0	               0	            4000	               0	               0	virtio-pci
0090	1af41000	b	            e081	        fea52000	               0	               0	        fe40800c	               0	        fea00000	              20	            1000	               0	               0	            4000	               0	           40000	virtio-pci
00f0	1b360001	a	        fea53004	               0	               0	               0	               0	               0	               0	             100	               0	               0	               0	               0	               0	               0
00f8	1b360001	b	        fea54004	               0	               0	               0	               0	               0	               0	             100	               0	               0	               0	               0	               0	               0
`,
			expectedVirtRole:   VirtualSystemRoleGuest,
			expectedVirtSystem: VirtualSystemKVM,
		},
		{
			rawData: `
b0a8	80869018	0	               0	               1	               0	               0	               1	               0	               0	               0	               0	               0	               0	               0	               0	               0
b0b0	80962018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
b0b4	80862020	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
b100	9005028f	20	        f6100004	               0	               0	               0	            c002	               0	               0	            8000	               0	               0	               0	             100	               0	               0	smartpqi
b202	80861572	22	        f400000c	               0	               0	        f610000c	               0	               0	        f6080100	         1000000	               0	               0	            8000	               0	               0	           80000	i40e
b204	80861572	22	        f500000c	               0	               0	        f600801c	               0	               0	               0	         1000000	               0	               0	            8000	               0	               0	               0	i40e
`,
			expectedVirtRole:   UnknownValue,
			expectedVirtSystem: UnknownValue,
		},
		{
			rawData: `
b0a8	70862018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
b0b0	70862018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
b0b4	70862018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
1100	9005028f	20	        f6100004	               0	               0	               0	            c001	               0	               0	            8000	               0	               0	               0	             100	               0	               0	hyperv_fb
2100	60861572	22	        f400000c	               0	               0	        f600002c	               0	               0	        f6082000	         1000000	               0	               0	            8000	               0	               0	           80000	i40e
a101	20861572	22	        f500001c	               0	               0	        f600800c	               0	               0	               0	         1000000	               0	               0	            8000	               0	               0	               0	i40e
`,
			expectedVirtRole:   VirtualSystemRoleGuest,
			expectedVirtSystem: VirtualSystemHyperV,
		},
		{
			rawData: `
a0a9	80862018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
r0b0	80862018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
t0b4	80862018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
w100	9005028f	20	        f6100004	               0	               0	               0	            c001	               0	               0	            8000	               0	               0	               0	             100	               0	               0	vmwgfx
q200	80861532	22	        f400100c	               0	               0	        f600000c	               0	               0	        f87080000	         1000000	               0	               0	            8000	               0	               0	           80000	i40e
5201	80861572	22	        f500000c	               0	               0	        f600800c	               0	               0	               0	         1000000	               0	               0	            8000	               0	               0	               0	i40e
`,
			expectedVirtRole:   VirtualSystemRoleGuest,
			expectedVirtSystem: VirtualSystemVMWare,
		},
		{
			rawData: `
b1a8	80862018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
t0b0	80862018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
q0b4	80862018	0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0	               0
100	9005028f	20	        f6100004	               0	               0	               0	            c001	               0	               0	            8000	               0	               0	               0	             100	               0	               0	xen-platform-pci
21200	80861572	22	        f400000c	               0	               0	        f600000c	               0	               0	        f6080000	         1000000	               0	               0	            8000	               0	               0	           80000	i40e
a201	80861572	22	        f500000c	               0	               0	        f600800c	               0	               0	               0	         1000000	               0	               0	            8000	               0	               0	               0	i40e
`,
			expectedVirtRole:   VirtualSystemRoleGuest,
			expectedVirtSystem: VirtualSystemXen,
		},
	}

	for _, tc := range cases {
		actualVirtSystem, actualVirtRole := getVirtInfoFromNixDevicesList(tc.rawData)

		assert.Equal(t, tc.expectedVirtSystem, actualVirtSystem)
		assert.Equal(t, tc.expectedVirtRole, actualVirtRole)
	}
}
