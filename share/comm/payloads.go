package comm

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// RequestTypeCheckPort request types sent by server to clients
	RequestTypeCheckPort            = "check_port"
	RequestTypeRunCmd               = "run_cmd"
	RequestTypeRefreshUpdatesStatus = "refresh_updates_status"
	RequestTypePutCapabilities      = "put_capabilities"
	RequestTypeCheckTunnelAllowed   = "check_tunnel_allowed"

	RequestTypeUpdateClientAttributes = "update_client_metadata"

	// RequestTypeCmdResult request types sent by clients to server
	RequestTypeCmdResult       = "cmd_result"
	RequestTypeUpdatesStatus   = "updates_status"
	RequestTypeSaveMeasurement = "save_measurement"
	RequestTypeUpload          = "upload"
	RequestTypeIPAddresses     = "ip_addresses"
	RequestTypeInventory       = "inventory"

	// RequestTypePing request types understood on both sides, client and server
	RequestTypePing = "ping"
)

type CheckPortRequest struct {
	HostPort string
	Timeout  time.Duration
}

func DecodeCheckPortRequest(b []byte) (*CheckPortRequest, error) {
	res := &CheckPortRequest{}
	if err := json.Unmarshal(b, res); err != nil {
		return nil, fmt.Errorf("failed to decode %T: %v", res, err)
	}
	return res, nil
}

type CheckPortResponse struct {
	Open   bool
	ErrMsg string
}

type RunCmdResponse struct {
	Pid       int
	StartedAt time.Time
}

type CheckTunnelAllowedRequest struct {
	Remote string
}

type CheckTunnelAllowedResponse struct {
	IsAllowed bool
}
