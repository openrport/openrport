package scriptRunner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

type ScriptRunner interface {
	Run(script string, recipients []string, subject string, body string) error
}

type runner struct {
	scriptTimeout time.Duration
}

func (r runner) Run(script string, recipients []string, subject string, body string) error {

	args := append([]string{subject}, recipients...)

	cmd := exec.Command(script, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	var errb bytes.Buffer
	cmd.Stderr = &errb

	if err = cmd.Start(); err != nil { //Use start, not run
		return err
	}

	_, err = io.WriteString(stdin, body)
	if err != nil {
		return err
	}

	err = stdin.Close()
	if err != nil {
		return err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		select {
		case <-time.After(r.scriptTimeout):
			err = cmd.Process.Kill()
			if err != nil {
				err = fmt.Errorf("error killing after timeout script: %v", err)
			} else {
				err = fmt.Errorf("script timeout")
			}

			cancelFunc()
		case <-ctx.Done():
		}
	}()

	go func() {
		err = cmd.Wait()
		cancelFunc()
	}()

	<-ctx.Done()
	if err != nil {
		return err
	}

	if errb.Len() > 0 {
		return fmt.Errorf("there is something on stderr: %v", errb.String())
	}

	return nil
}

func NewScriptRunner(scriptTimeout time.Duration) ScriptRunner {
	return runner{
		scriptTimeout: scriptTimeout,
	}
}
