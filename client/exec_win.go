//+build windows

package chclient

import (
	"context"
	"errors"

	"github.com/cloudradar-monitoring/rport/share/comm"
)

func (c *Client) HandleRunCmdRequest(ctx context.Context, reqPayload []byte) (*comm.RunCmdResponse, error) {
	return nil, errors.New("remote command execution not supported")
}
