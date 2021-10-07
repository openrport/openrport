// +build windows

package winapi

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modiphlpapi = windows.NewLazySystemDLL("iphlpapi.dll")

	procGetAdaptersAddresses = modiphlpapi.NewProc("GetAdaptersAddresses")
	procGetIfEntry2          = modiphlpapi.NewProc("GetIfEntry2")
)

const (
	IF_MAX_PHYS_ADDRESS_LENGTH = 32
)

// IP_ADAPTER_ADDRESSES_LH
// https://docs.microsoft.com/ru-ru/windows/win32/api/iptypes/ns-iptypes-_ip_adapter_addresses_lh
type IPAdapterAddresses struct {
	Length                uint32
	IfIndex               uint32
	Next                  *IPAdapterAddresses
	AdapterName           *byte
	FirstUnicastAddress   *windows.IpAdapterUnicastAddress
	FirstAnycastAddress   *windows.IpAdapterAnycastAddress
	FirstMulticastAddress *windows.IpAdapterMulticastAddress
	FirstDNSServerAddress *windows.IpAdapterDnsServerAdapter
	DNSSuffix             *uint16
	Description           *uint16
	FriendlyName          *uint16
	PhysicalAddress       [syscall.MAX_ADAPTER_ADDRESS_LENGTH]byte
	PhysicalAddressLength uint32
	Flags                 uint32
	Mtu                   uint32
	IfType                uint32
	OperStatus            uint32
	Ipv6IfIndex           uint32
	ZoneIndices           [16]uint32
	FirstPrefix           *windows.IpAdapterPrefix
	TransmitLinkSpeed     uint64
	ReceiveLinkSpeed      uint64
	/* more fields might be present here. */
}

type NET_LUID NET_LUID_LH
type NET_LUID_LH struct {
	Value uint64
}

type NET_IF_GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// GUID
type NET_IF_NETWORK_GUID NET_IF_GUID

// MIB_IF_ROW2
// https://docs.microsoft.com/ru-ru/windows/win32/api/netioapi/ns-netioapi-_mib_if_row2
type MibIfRow2 struct {
	InterfaceLuid               NET_LUID
	InterfaceIndex              uint32
	InterfaceGuid               NET_IF_GUID
	Alias                       [windows.MAX_ADAPTER_NAME_LENGTH + 1]uint16
	Description                 [windows.MAX_ADAPTER_NAME_LENGTH + 1]uint16
	PhysicalAddressLength       uint32
	PhysicalAddress             [IF_MAX_PHYS_ADDRESS_LENGTH]byte
	PermanentPhysicalAddress    [IF_MAX_PHYS_ADDRESS_LENGTH]byte
	Mtu                         uint32
	Type                        uint32
	TunnelType                  int32
	MediaType                   int32
	PhysicalMediumType          int32
	AccessType                  int32
	DirectionType               int32
	InterfaceAndOperStatusFlags bool
	OperStatus                  int32
	AdminStatus                 int32
	MediaConnectState           int32
	NetworkGuid                 NET_IF_NETWORK_GUID
	ConnectionType              int32
	padding1                    [pad0for64_4for32]byte
	TransmitLinkSpeed           uint64
	ReceiveLinkSpeed            uint64
	InOctets                    uint64
	InUcastPkts                 uint64
	InNUcastPkts                uint64
	InDiscards                  uint64
	InErrors                    uint64
	InUnknownProtos             uint64
	InUcastOctets               uint64
	InMulticastOctets           uint64
	InBroadcastOctets           uint64
	OutOctets                   uint64
	OutUcastPkts                uint64
	OutNUcastPkts               uint64
	OutDiscards                 uint64
	OutErrors                   uint64
	OutUcastOctets              uint64
	OutMulticastOctets          uint64
	OutBroadcastOctets          uint64
	OutQLen                     uint64
}

func (a *IPAdapterAddresses) GetInterfaceName() string {
	return syscall.UTF16ToString((*(*[10000]uint16)(unsafe.Pointer(a.FriendlyName)))[:])
}

// GetAdaptersAddresses returns a list of IP adapter and address
// structures. The structure contains an IP adapter and flattened
// multiple IP addresses including unicast, anycast and multicast
// addresses.
func GetAdaptersAddresses() ([]*IPAdapterAddresses, error) {
	var b []byte
	l := uint32(15000) // recommended initial size
	for {
		b = make([]byte, l)
		var err error
		r0, _, _ := syscall.Syscall6(
			procGetAdaptersAddresses.Addr(),
			5,
			uintptr(syscall.AF_UNSPEC),
			uintptr(windows.GAA_FLAG_INCLUDE_PREFIX),
			uintptr(0),
			uintptr(unsafe.Pointer((*IPAdapterAddresses)(unsafe.Pointer(&b[0])))),
			uintptr(unsafe.Pointer(&l)),
			0,
		)

		if r0 == 0 {
			if l == 0 {
				return nil, nil
			}
			break
		} else {
			err = syscall.Errno(r0)
		}

		if err.(syscall.Errno) != syscall.ERROR_BUFFER_OVERFLOW {
			return nil, os.NewSyscallError("GetAdaptersAddresses", err)
		}

		if l <= uint32(len(b)) {
			return nil, os.NewSyscallError("GetAdaptersAddresses", err)
		}
	}
	var aas []*IPAdapterAddresses
	for aa := (*IPAdapterAddresses)(unsafe.Pointer(&b[0])); aa != nil; aa = aa.Next {
		aas = append(aas, aa)
	}
	return aas, nil
}

// https://docs.microsoft.com/en-us/windows/win32/api/netioapi/nf-netioapi-getifentry2
// On input, the InterfaceLuid or the InterfaceIndex member of the MibIfRow2 must be set to the interface for which to retrieve information.
func GetIfEntry2(row *MibIfRow2) error {
	r0, _, _ := syscall.Syscall(procGetIfEntry2.Addr(), 1, uintptr(unsafe.Pointer(row)), 0, 0)
	if r0 != 0 {
		err := syscall.Errno(r0)
		return os.NewSyscallError("GetIfEntry2", err)
	}

	return nil
}
