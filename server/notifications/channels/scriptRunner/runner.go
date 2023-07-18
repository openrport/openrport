package scriptRunner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

func RunCancelableScript(ctx context.Context, script string, body string) (string, error) {

	cmd := exec.CommandContext(ctx, script)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	var okb bytes.Buffer
	cmd.Stdout = &okb

	var errb bytes.Buffer
	cmd.Stderr = &errb

	if err = cmd.Start(); err != nil { //Use start, not run
		return "", err
	}

	_, err = io.WriteString(stdin, body)
	if err != nil {
		return "", err
	}

	err = stdin.Close()
	if err != nil {
		return "", err
	}

	internalCtx, internalCancelFunc := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			err = cmd.Process.Kill()
			if err != nil {
				err = fmt.Errorf("killing of the script failed, script killed because of ctx cancel: %v", err)
			} else {
				err = fmt.Errorf("script killed because of ctx cancel")
			}

			time.Sleep(time.Millisecond * 200)
			internalCancelFunc()
		case <-internalCtx.Done():
		}
	}()

	go func() {
		err = cmd.Wait()
		internalCancelFunc()
	}()

	<-internalCtx.Done()

	// this err will be set by the goroutine closures above
	if err != nil {
		return okb.String(), err
	}

	if errb.Len() > 0 {
		return okb.String(), fmt.Errorf("there is something on stderr: %v", errb.String())
	}

	return okb.String(), nil
}
