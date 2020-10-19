package chshare

import (
	"encoding/json"
	"fmt"
)

// ConnectionRequest represents configuration options when initiating client-server connection
type ConnectionRequest struct {
	Version  string
	ID       string
	Name     string
	OS       string
	OSArch   string
	OSFamily string
	OSKernel string
	Hostname string
	IPv4     []string
	IPv6     []string
	Tags     []string
	Remotes  []*Remote
}

func DecodeConnectionRequest(b []byte) (*ConnectionRequest, error) {
	c := &ConnectionRequest{}
	err := json.Unmarshal(b, c)
	if err != nil {
		return nil, fmt.Errorf("Invalid JSON config")
	}
	return c, nil
}

func EncodeConnectionRequest(c *ConnectionRequest) ([]byte, error) {
	return json.Marshal(c)
}
