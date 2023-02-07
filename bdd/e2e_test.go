package bdd_test

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServerStarts(t *testing.T) {
	rd, err := run(t, "..", "./cmd/rportd/main.go")
	assert.Nil(t, err)
	assert.NotNil(t, rd)
	assert.Nil(t, rd.ProcessState)
	time.Sleep(1 * time.Second)
	assert.Nil(t, rd.ProcessState)
	assert.Nil(t, rd.Process.Kill())
}

func TestClientConnects(t *testing.T) {

	rd, err := run(t, "..", "./cmd/rportd/main.go")
	assert.Nil(t, err)
	_, err = run(t, "..", "./cmd/rport/main.go")
	assert.Nil(t, err)

	assert.Nil(t, rd.ProcessState)
	assert.Nil(t, rd.ProcessState)

	assert.Fail(t, "work in progress...")
}

func run(t *testing.T, pwd string, cmd string) (*exec.Cmd, error) {
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

	errChan := make(chan error, 1)
	startChan := make(chan any, 1)

	go func() {
		for errOut.Scan() {
			errTxt := errOut.Text()
			fmt.Println(cmd, "---", "error:", errTxt)
			assert.Fail(t, "daemon logged error")
			errChan <- errors.New(errTxt)
			rd.Process.Kill()
		}
	}()

	go func() {
		for out.Scan() {
			fmt.Println(cmd, "---", out.Text())
			startChan <- true
		}

	}()

	select {
	case err = <-errChan:
		return nil, err

	case <-startChan:
		return rd, nil
	}

}
