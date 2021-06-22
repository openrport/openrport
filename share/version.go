package chshare

//ProtocolVersion of rport. When backwards
//incompatible changes are made, this will
//be incremented to signify a protocol
//mismatch.
const ProtocolVersion = "rport-v1"

// BuildVersion represents a current build version. It can be overridden by CI workflow.
var BuildVersion = SourceVersion
var SourceVersion = "0.0.0-src"
