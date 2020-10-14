//+build windows

package chclient

import "errors"

func (c *Client) HandleRunCmdRequest(ctx context.Context, reqPayload []byte) (*comm.RunCmdResponse, error) {
	return nil, errors.New("remote command execution not supported")
}
