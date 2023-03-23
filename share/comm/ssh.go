// Package comm is responsible for sharing logic to handle communication between a server and clients.
package comm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/realvnc-labs/rport/share/logger"
)

type TimeoutError struct {
	error
}

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
func SendRequestAndGetResponse(conn ssh.Conn, reqType string, req, successRespDest interface{}, l *logger.Logger) error {
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
			l.Debugf("failed to unmarshal: respBytes: %s", string(respBytes))
			l.Debugf("%#v", successRespDest)
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

type requestResponse struct {
	ok       bool
	response []byte
	err      error
}

func SendRequestWithTimeout(parentCtx context.Context, conn ssh.Conn, name string, wantReply bool, payload []byte, timeout time.Duration, l *logger.Logger) (bool, []byte, error) {
	var (
		ok       bool
		response []byte
		err      error
	)

	if conn == nil {
		return false, nil, errors.New("cannot send request when conn is nil")
	}

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	ch := make(chan requestResponse, 1)

	go func() {
		ok, response, err = conn.SendRequest(name, wantReply, payload)
		select {
		default:
			ch <- requestResponse{
				ok:       ok,
				response: response,
				err:      err,
			}
		case <-ctx.Done():
			l.Debugf("send canceled")
			return
		}
	}()

	reqTimeout := time.NewTimer(timeout)
	select {
	case res := <-ch:
		reqTimeout.Stop()
		return res.ok, res.response, res.err
	case <-reqTimeout.C:
		return false, nil, TimeoutError{fmt.Errorf("conn.SendRequest(%s), timeout %s exceeded", name, timeout)}
	}
}

const DefaultMaxRetryAttempts = 3
const DefaultRetryWaitJitterMilliSeconds = 500

type retryCheckerFn func(err error) (shouldRetry bool)

func WithRetry[R any](retryAbleFn func() (result R, err error), canRetryFn retryCheckerFn, minRetryWaitDuration time.Duration, label string, l *logger.Logger) (result R, err error) {
	for r := 0; r < DefaultMaxRetryAttempts; r++ {
		attempt := r + 1
		// l.Debugf("%s: attempt %d", label, attempt)
		if attempt > 1 {
			// backoff with some jitter
			delay := (minRetryWaitDuration * time.Duration(r*r)) + time.Duration(rand.Intn(DefaultRetryWaitJitterMilliSeconds))*time.Millisecond
			l.Debugf("%s: attempt %d failed. will sleep for: %0.2f seconds", label, attempt, delay.Seconds())
			time.Sleep(delay)
		}
		result, err = retryAbleFn()
		if err != nil {
			l.Debugf("%s: attempt %d err = %+v\n", label, attempt, err)
			if !canRetryFn(err) {
				// non retryable err
				l.Debugf("%s: attempt %d non-retryable err %v", label, attempt, err)
				return result, err
			}
			continue
		}
		// success
		return result, nil
	}

	return result, err
}
