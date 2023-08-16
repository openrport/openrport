package scriptRunner

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

func RunCancelableScript(ctx context.Context, workingDir string, script string, body string) (string, error) {

	cmd := exec.CommandContext(ctx, script)

	cmd.Dir = workingDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	var okb Buffer
	defer okb.CollectiongDone()
	cmd.Stdout = &okb

	var errb Buffer
	defer errb.CollectiongDone()
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
	// log.Println("pid: ", cmd.Process.Pid)

	killCh := make(chan error, 10)
	doneCh := make(chan error, 10)

	process, processDone := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			// log.Println("pid: ", cmd.Process.Pid)
			err := cmd.Process.Kill()
			if err != nil {
				killCh <- fmt.Errorf("killing of the script failed, script killed because of ctx cancel: %v", err)
			} else {
				killCh <- fmt.Errorf("script killed because of ctx cancel")
			}

		case <-process.Done():
		}
		close(killCh)
	}()

	go func() {
		err := cmd.Wait()
		if err != nil {
			doneCh <- fmt.Errorf("process error: %v", err)
		}
		close(doneCh)
		processDone()
	}()

	errs := []string{}

	select {
	case <-ctx.Done():
		select {
		case <-process.Done():
			for err := range doneCh {
				errs = append(errs, err.Error())
			}
		case <-time.After(time.Millisecond * 200):
			for err := range killCh {
				errs = append(errs, err.Error())
			}
			errs = append(errs, fmt.Errorf("process hangs, ignoring").Error())
		}
	case <-process.Done():
		for err := range doneCh {
			errs = append(errs, err.Error())
		}
	}

	errSummary := strings.Join(errs, ", ")

	if errSummary != "" {
		return okb.String(), fmt.Errorf("errors running script: %v, stdErr: %v", errSummary, errb.String())
	}

	if errb.Len() > 0 {
		return okb.String(), fmt.Errorf("there is something on stderr: %v", errb.String())
	}

	return okb.String(), nil
}
