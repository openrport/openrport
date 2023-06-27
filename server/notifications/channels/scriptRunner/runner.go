package scriptRunner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
)

func RunCancelableScript(ctx context.Context, script string, body string) error {

	cmd := exec.Command(script)

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

	internalCtx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			err = cmd.Process.Kill()
			if err != nil {
				err = fmt.Errorf("killing of the script failed, script killed because of ctx cancel: %v", err)
			} else {
				err = fmt.Errorf("script killed because of ctx cancel")
			}

			cancelFunc()
		case <-internalCtx.Done():
		}
	}()

	go func() {
		err = cmd.Wait()
		cancelFunc()
	}()

	<-internalCtx.Done()
	if err != nil {
		return err
	}

	if errb.Len() > 0 {
		return fmt.Errorf("there is something on stderr: %v", errb.String())
	}

	return nil
}
