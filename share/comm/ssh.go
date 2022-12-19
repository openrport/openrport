// Package comm is responsible for sharing logic to handle communication between a server and clients.
package comm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

type TimeoutError error

// ReplyError sends a failure response with a given error message if not nil to a given request.
func ReplyError(log *logger.Logger, req *ssh.Request, err error) {
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
func ReplySuccessJSON(log *logger.Logger, req *ssh.Request, resp interface{}) {
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

// SendRequestAndGetResponse sends a given request, parses a returned response and stores a success result in a given destination value.
// Returns an error on a failure response or if an error happen. Error will be ClientError type if the error is a client error.
// Both request and response are expected to be JSON.
func SendRequestAndGetResponse(conn ssh.Conn, reqType string, req, successRespDest interface{}) error {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request %T: %v", req, err)
	}

	ok, respBytes, err := conn.SendRequest(reqType, true, reqBytes)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if !ok {
		return NewClientError(fmt.Errorf("client error: %s", respBytes))
	}

	if successRespDest != nil {
		if err := json.Unmarshal(respBytes, successRespDest); err != nil {
			return NewClientError(fmt.Errorf("invalid client response format: failed to decode response into %T: %v", successRespDest, err))
		}
	}

	return nil
}

type ClientError struct {
	err error
}

func NewClientError(err error) *ClientError {
	return &ClientError{err: err}
}

func (e *ClientError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func SendRequestWithTimeout(conn ssh.Conn, name string, wantReplay bool, payload []byte, timeout time.Duration) (bool, []byte, error) {
	var (
		ok       bool
		response []byte
		err      error
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan bool, 1)
	go func() {
		ok, response, err = conn.SendRequest(name, wantReplay, payload)
		select {
		default:
			ch <- true
		case <-ctx.Done():
			return
		}
	}()
	reqTimeout := time.NewTimer(timeout)
	defer reqTimeout.Stop()
	select {
	case <-ch:
		return ok, response, err
	case <-reqTimeout.C:
		return false, nil, TimeoutError(fmt.Errorf("conn.SendRequest(%s), timeout %s exceeded", name, timeout))
	}
}
