package comm

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// request types sent by server to clients
	RequestTypeCheckPort  = "check_port"
	RequestTypeRunCmd     = "run_cmd"
	RequestTypeCreateFile = "create_file"

	// request types sent by clients to server
	RequestTypePing          = "ping"
	RequestTypeCmdResult     = "cmd_result"
	RequestTypeUpdatesStatus = "updates_status"
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

type CreateFileResponse struct {
	FilePath   string
	Sha256Hash string
	CreatedAt  time.Time
}
