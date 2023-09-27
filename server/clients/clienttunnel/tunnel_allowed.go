package clienttunnel

import (
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/openrport/openrport/share/comm"
	"github.com/openrport/openrport/share/logger"
)

func IsAllowed(remote string, conn ssh.Conn, l *logger.Logger) (bool, error) {
	req := &comm.CheckTunnelAllowedRequest{
		Remote: remote,
	}
	resp := &comm.CheckTunnelAllowedResponse{}
	err := comm.SendRequestAndGetResponse(conn, comm.RequestTypeCheckTunnelAllowed, req, resp, l)
	if err != nil {
		if strings.Contains(err.Error(), "unknown request") {
			return true, nil
		}
		return false, err
	}

	return resp.IsAllowed, nil
}
