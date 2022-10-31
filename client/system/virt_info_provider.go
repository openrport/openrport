package system

import (
	"regexp"
	"strings"
)

const (
	VirtualSystemHyperV    = "HyperV"
	VirtualSystemVMWare    = "VMware"
	VirtualSystemKVM       = "KVM"
	VirtualSystemXen       = "Xen"
	VirtualSystemRoleGuest = "guest"
	VirtualSystemRoleHost  = "host"
)

const UnknownValue = "unknown"

/*
This method interprets result of powerShell Get-Service output as following:
Get-Service | findstr vmcompute detect if a machine is a HyperV host.
This would result in "os_virtualization_system": "HyperV", "os_virtualization_role":"guest"
Get-Service|findstr "Running.*vmicheartbeat" detects if a machine is a Hyper-V guest.
Get-service|findstr "Running.*VMTools" detects if a machine is VMware guest.
Get-service|findstr "Running.*QEMU-GA" detects if a machine is KVM guest.
*/
func getVirtInfoFromPowershellServicesList(rawServicesList string) (virtSystem, virtRole string) {
	if strings.Contains(rawServicesList, "vmcompute") {
		virtSystem = VirtualSystemHyperV
		virtRole = VirtualSystemRoleHost

		return virtSystem, virtRole
	}

	regexToVirtInfoMapping := map[string]struct {
		virtSystem string
		virtRole   string
	}{
		`Running.*vmicheartbeat`: {
			virtSystem: VirtualSystemHyperV,
			virtRole:   VirtualSystemRoleGuest,
		},
		`Running.*VMTools`: {
			virtSystem: VirtualSystemVMWare,
			virtRole:   VirtualSystemRoleGuest,
		},
		`Running.*QEMU-GA`: {
			virtSystem: VirtualSystemKVM,
			virtRole:   VirtualSystemRoleGuest,
		},
	}

	for regexStr, virtInfo := range regexToVirtInfoMapping {
		rx := regexp.MustCompile(regexStr)
		if rx.MatchString(rawServicesList) {
			return virtInfo.virtSystem, virtInfo.virtRole
		}
	}

	return UnknownValue, UnknownValue
}

/*
Parses the output of /proc/bus/pci/devices on nix systems as following:
If grep -c hyperv_fb /proc/bus/pci/devices > 0 the Linux system is a HyperV guest.
If grep -c vmwgfx /proc/bus/pci/devices > 0 the Linux system is a Vmware guest.
If grep -c virtio-pci /proc/bus/pci/devices > 0 the Linux system is a KVM guest.
If grep -c xen-platform-pci /proc/bus/pci/devices > 0 the Linux system is a Xen guest.
*/
func getVirtInfoFromNixDevicesList(rawDevicesList string) (virtSystem, virtRole string) {
	devicesInfoExpectations := []struct {
		expectedSubstr string
		virtSystem     string
		virtRole       string
	}{
		{
			expectedSubstr: "hyperv_fb",
			virtSystem:     VirtualSystemHyperV,
			virtRole:       VirtualSystemRoleGuest,
		},
		{
			expectedSubstr: "vmwgfx",
			virtSystem:     VirtualSystemVMWare,
			virtRole:       VirtualSystemRoleGuest,
		},
		{
			expectedSubstr: "virtio-pci",
			virtSystem:     VirtualSystemKVM,
			virtRole:       VirtualSystemRoleGuest,
		},
		{
			expectedSubstr: "xen-platform-pci",
			virtSystem:     VirtualSystemXen,
			virtRole:       VirtualSystemRoleGuest,
		},
	}

	for _, exp := range devicesInfoExpectations {
		if strings.Count(rawDevicesList, exp.expectedSubstr) > 0 {
			return exp.virtSystem, exp.virtRole
		}
	}

	return UnknownValue, UnknownValue
}
