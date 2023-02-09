package helpers

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/KonradKuznicki/must"
	"github.com/stretchr/testify/assert"
)

func CallAPIGET(t *testing.T, requestURL string) string {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	assert.Nil(t, err)
	req.SetBasicAuth("admin", "foobaz")
	res, err := client.Do(req)
	assert.Nil(t, err)
	data := string(must.Must(io.ReadAll(res.Body)))
	log.Println(data)
	return data
}

func StartClientAndServerAndWaitForConnection(t *testing.T) (*exec.Cmd, *exec.Cmd) {
	rd, rdOutChan, _ := Run(t, "", "../../cmd/rportd/main.go")
	defer func() {
		rd.Process.Kill()
	}()

	err := WaitForText(rdOutChan, "API Listening") // wait for server to initialize and boot
	assert.Nil(t, err)

	rc, rcOutChan, _ := Run(t, "", "../../cmd/rport/main.go")

	err = WaitForText(rcOutChan, "info: client: Connected") // wait for client to connect
	assert.Nil(t, err)

	return rd, rc
}

func WaitForText(ch chan string, txt string) error {
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
	case <-time.After(time.Second * 15):
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
	}()

	go func() {
		for out.Scan() {
			text := out.Text()
			fmt.Println(cmd, "---", text)
			startChan <- text
		}

	}()

	select {
	case <-errChan:
		return rd, startChan, errChan

	case <-startChan:
		return rd, startChan, errChan
	}

}
