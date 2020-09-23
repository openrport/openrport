package comm

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	RequestTypeCheckPort = "check_port"
)

type CheckPortRequest struct {
	HostPort string
	Timeout  time.Duration
}

func DecodeCheckPortRequest(b []byte) (*CheckPortRequest, error) {
	res := &CheckPortRequest{}
	if err := json.Unmarshal(b, res); err != nil {
		return nil, fmt.Errorf("failed to unmarshall %T: %v", res, err)
	}
	return res, nil
}

type CheckPortResponse struct {
	Open   bool
	ErrMsg string
}
