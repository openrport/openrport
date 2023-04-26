package helpers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func StartClientAndServerAndWaitForConnection(ctx context.Context, t *testing.T, projectRoot string) (*exec.Cmd, *exec.Cmd) {

	internalCtx, cancelFn := context.WithCancel(ctx)

	rd, rdOutChan, rdErrChan := Run(t, "", path.Join(projectRoot, "cmd/rportd/main.go"))
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

	rc, rcOutChan, rcErrChan := Run(t, "", path.Join(projectRoot, "cmd/rport/main.go"))
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
	time.Sleep(time.Millisecond * 100)
	if rd.ProcessState != nil || rc.ProcessState != nil {
		assert.Fail(t, "daemons didn't start")
	}
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
	// there can be an error, but I don't care and want to silence the linter
	log.Println(err)
}

func FindProjectRoot(t *testing.T) string {
	getwd, err := os.Getwd()
	if err != nil {
		assert.Failf(t, "couldn't find project root: %v", err.Error())
		return ""
	}

	parts := append([]string{filepath.VolumeName(getwd)}, strings.Split(getwd, string(filepath.Separator))...)

	parts = parts[:len(parts)-2]
	for i := len(parts); i > 0; i-- {
		left := parts[:i]
		basePath := filepath.Join(left...)
		testPath := filepath.Join(basePath, "go.mod")
		_, err := os.Stat(testPath)
		if err == nil {
			t.Log("found root", basePath)
			return basePath
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		t.Log("error checking path ", err)
	}

	assert.Failf(t, "couldn't find project root: %v", err.Error())
	return ""
}
