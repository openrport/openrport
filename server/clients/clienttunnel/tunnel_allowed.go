package clienttunnel

import (
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/share/comm"
)

func IsAllowed(remote string, conn ssh.Conn) (bool, error) {
	req := &comm.CheckTunnelAllowedRequest{
		Remote: remote,
	}
	resp := &comm.CheckTunnelAllowedResponse{}
	err := comm.SendRequestAndGetResponse(conn, comm.RequestTypeCheckPort, req, resp)
	if err != nil {
		return false, err
	}

	return resp.IsAllowed, nil
}
