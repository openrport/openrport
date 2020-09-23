// Package responsible for sharing logic to handle communication between a server and clients.
package comm

import (
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/ssh"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

// ReplyError sends a failure response with a given error message if not nil to a given request.
func ReplyError(log *chshare.Logger, req *ssh.Request, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	if replyErr := req.Reply(false, []byte(errMsg)); replyErr != nil {
		log.Errorf("Failed to reply an error response: %v", replyErr)
	}
}

// ReplySuccessJSON sends a success response with a given value as JSON to a given request.
// Response expected to be a value that can be encoded into JSON, otherwise - a failure will be replied.
func ReplySuccessJSON(log *chshare.Logger, req *ssh.Request, resp interface{}) {
	respBytes, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("Failed to encode success response %T: %v", resp, err)
		ReplyError(log, req, err)
		return
	}

	if err = req.Reply(true, respBytes); err != nil {
		log.Errorf("Failed to reply a success response %T: %v", resp, err)
	}
}

// HandleSSHRequestJSON sends a given request, parses the returned response and stores the result in a given destination value.
// Both request and response are expected to be JSON. Returns an error on a failure response or if an error happen.
func HandleSSHRequestJSON(conn ssh.Conn, reqType string, req, respDest interface{}) error {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode %T: %v", req, err)
	}

	ok, respBytes, err := conn.SendRequest(reqType, true, reqBytes)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if !ok {
		return fmt.Errorf("received an error response: %s", respBytes)
	}

	if err := json.Unmarshal(respBytes, respDest); err != nil {
		return fmt.Errorf("failed to decode success response %T: %v", respDest, err)
	}

	return nil
}
