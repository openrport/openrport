package chshare

type Capabilities struct {
	ServerVersion string
	Monitoring    int
}

func NewCapabilities() *Capabilities {
	return &Capabilities{
		ServerVersion: BuildVersion,
		Monitoring:    1,
	}
}
