package helpers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func StartClientAndServerAndWaitForConnection(ctx context.Context, t *testing.T) (*exec.Cmd, *exec.Cmd) {

	internalCtx, cancelFn := context.WithCancel(ctx)

	rd, rdOutChan, rdErrChan := Run(t, "", "../../cmd/rportd/main.go")
	go func() {
		for line := range rdErrChan {
			if strings.Contains(line, "go: downloading") {
				continue
			}
			assert.Fail(t, "server errors on stdErr")
			LogAndIgnore(rd.Process.Kill())
			cancelFn()
		}
	}()

	err := WaitForText(internalCtx, rdOutChan, "API Listening") // wait for server to initialize and boot - takes looooong time
	assert.Nil(t, err)

	rc, rcOutChan, rcErrChan := Run(t, "", "../../cmd/rport/main.go")
	go func() {
		for line := range rcErrChan {
			if strings.Contains(line, "go: downloading") {
				continue
			}
			assert.Fail(t, "client errors on stdErr")
			LogAndIgnore(rc.Process.Kill())
			cancelFn()
		}
	}()

	err = WaitForText(internalCtx, rcOutChan, "info: client: Connected") // wait for client to connect - sloooooow - needs to compile...
	assert.Nil(t, err)

	return rd, rc
}

func WaitForText(ctx context.Context, ch chan string, txt string) error {
	txtMatched := make(chan bool, 1)
	go func() {
		for lineOut := range ch {
			if strings.Contains(lineOut, txt) {
				txtMatched <- true
				return
			}
		}
	}()

	select {
	case <-txtMatched:
		return nil
	case <-ctx.Done():
		return errors.New("timeout waiting for text: " + txt)
	}

}

func Run(t *testing.T, pwd string, cmd string) (*exec.Cmd, chan string, chan string) {
	rd := exec.Command("go", "run", cmd)
	rd.Dir = pwd

	outPipe, err := rd.StdoutPipe()
	assert.Nil(t, err)

	out := bufio.NewScanner(outPipe)
	assert.Nil(t, err)

	errPipe, err := rd.StderrPipe()
	assert.Nil(t, err)

	errOut := bufio.NewScanner(errPipe)
	assert.Nil(t, err)

	err = rd.Start()
	assert.Nil(t, err)

	errChan := make(chan string, 1000)
	startChan := make(chan string, 1000)

	go func() {
		for errOut.Scan() {
			errTxt := errOut.Text()

			fmt.Println(cmd, "---", "error:", errTxt)
			// assert.Fail(t, "daemon logged error")
			errChan <- errTxt
			// rd.Process.Kill()
		}
		close(errChan)
	}()

	go func() {
		for out.Scan() {
			text := out.Text()
			fmt.Println(cmd, "---", text)
			startChan <- text
		}
		close(startChan)
	}()

	return rd, startChan, errChan

}

func LogAndIgnore(err error) {
	// yolo :)
	// there can be an error, but I don't care and want to silence the linter
	log.Println(err)
}
