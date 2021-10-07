// +build windows

package winapi

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// WINHTTP_AUTOPROXY_OPTIONS
// https://docs.microsoft.com/en-us/windows/win32/api/winhttp/ns-winhttp-winhttp_autoproxy_options
const (
	// Attempt to automatically discover the URL of the PAC file using both DHCP and DNS queries to the local network.
	WINHTTP_AUTOPROXY_AUTO_DETECT = 0x00000001
	// Download the PAC file from the URL specified by lpszAutoConfigUrl in the WINHTTP_AUTOPROXY_OPTIONS structure.
	WINHTTP_AUTOPROXY_CONFIG_URL = 0x00000002
	// Use DHCP to locate the proxy auto-configuration file.
	WINHTTP_AUTO_DETECT_TYPE_DHCP = 0x00000001
	// Use DNS to attempt to locate the proxy auto-configuration file at a well-known location on the domain of the local computer.
	WINHTTP_AUTO_DETECT_TYPE_DNS_A = 0x00000002
	// Resolves all host names directly without a proxy.
	WINHTTP_ACCESS_TYPE_NO_PROXY = 0x00000001
)

var (
	winhttpDLL                             = windows.NewLazySystemDLL("winhttp.dll")
	procHttpOpen                           = winhttpDLL.NewProc("WinHttpOpen")
	procHttpCloseHandle                    = winhttpDLL.NewProc("WinHttpCloseHandle")
	procHttpSetTimeouts                    = winhttpDLL.NewProc("WinHttpSetTimeouts")
	procHttpGetProxyForUrl                 = winhttpDLL.NewProc("WinHttpGetProxyForUrl")
	procHttpGetIEProxyConfigForCurrentUser = winhttpDLL.NewProc("WinHttpGetIEProxyConfigForCurrentUser")
	procHttpGetDefaultProxyConfiguration   = winhttpDLL.NewProc("WinHttpGetDefaultProxyConfiguration")
)

func HttpOpen(pszAgentW *uint16, dwAccessType uint32, pszProxyW *uint16, pszProxyBypassW *uint16, dwFlags uint32) (HInternet, error) {
	if err := procHttpOpen.Find(); err != nil {
		return 0, err
	}
	r, _, err := procHttpOpen.Call(
		uintptr(unsafe.Pointer(pszAgentW)),
		uintptr(dwAccessType),
		uintptr(unsafe.Pointer(pszProxyW)),
		uintptr(unsafe.Pointer(pszProxyBypassW)),
		uintptr(dwFlags),
	)
	if r == 0 {
		return 0, err
	}
	return HInternet(r), nil

}

func SetTimeouts(hInternet HInternet, resolveTimeout int, connectTimeout int, sendTimeout int, receiveTimeout int) error {
	if err := procHttpSetTimeouts.Find(); err != nil {
		return err
	}
	r, _, err := procHttpSetTimeouts.Call(
		uintptr(hInternet),
		uintptr(resolveTimeout),
		uintptr(connectTimeout),
		uintptr(sendTimeout),
		uintptr(receiveTimeout))
	if r == 1 {
		return nil
	}
	return err
}

func HttpCloseHandle(hInternet HInternet) error {
	if err := procHttpCloseHandle.Find(); err != nil {
		return err
	}
	r, _, err := procHttpCloseHandle.Call(uintptr(hInternet))
	if r == 1 {
		return nil
	}

	return err
}

func HttpGetProxyForUrl(hInternet HInternet, targetUrl string, pAutoProxyOptions *HttpAutoProxyOptions) (*HttpProxyInfo, error) {
	if err := procHttpGetProxyForUrl.Find(); err != nil {
		return nil, err
	}

	targetUrlPtr, _ := windows.UTF16PtrFromString(targetUrl)
	p := new(HttpProxyInfo)
	r, _, err := procHttpGetProxyForUrl.Call(
		uintptr(hInternet),
		uintptr(unsafe.Pointer(targetUrlPtr)),
		uintptr(unsafe.Pointer(pAutoProxyOptions)),
		uintptr(unsafe.Pointer(p)))

	if r == 1 {
		return p, nil
	}
	return nil, err
}

func HttpGetDefaultProxyConfiguration() (*HttpProxyInfo, error) {
	pInfo := new(HttpProxyInfo)
	if err := procHttpGetDefaultProxyConfiguration.Find(); err != nil {
		return nil, err
	}
	r, _, err := procHttpGetDefaultProxyConfiguration.Call(uintptr(unsafe.Pointer(pInfo)))
	if r == 1 {
		return pInfo, nil
	}
	return nil, err
}

func HttpGetIEProxyConfigForCurrentUser() (*HttpCurrentUserIEProxyConfig, error) {
	if err := procHttpGetIEProxyConfigForCurrentUser.Find(); err != nil {
		return nil, err
	}
	p := new(HttpCurrentUserIEProxyConfig)
	r, _, err := procHttpGetIEProxyConfigForCurrentUser.Call(uintptr(unsafe.Pointer(p)))
	if r == 1 {
		return p, nil
	}

	return nil, err
}
