package chshare

// ProtocolVersion of rport. When backwards
// incompatible changes are made, this will
// be incremented to signify a protocol
// mismatch.
const ProtocolVersion = "rport-v1"

// BuildVersion represents a current build version. It can be overridden by CI workflow.
var BuildVersion = "0.0.0-src"

// SourceVersion represents a default build version that is used for binaries built from sources.
var SourceVersion = "0.0.0-src"

// MonitoringVersion represents the current version of monitoring capability. 0 means no monitoring available.
const MonitoringVersion = 1

// IPAddressesVersion represents the current version of IPAddresses fetching. 0 means no IPAddress fetching available.
const IPAddressesVersion = 1
