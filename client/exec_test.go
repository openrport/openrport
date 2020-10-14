package chclient

import (
	"log"
	"os"
	"os/exec"
	"sync"
)

type CmdExecutorMock struct {
	ReturnPID      int
	ReturnStartErr error
	ReturnWaitErr  error
	ReturnStdOut   []string
	ReturnStdErr   []string

	wg sync.WaitGroup
}

func NewCmdExecutorMock() *CmdExecutorMock {
	return &CmdExecutorMock{}
}

func (e *CmdExecutorMock) Start(cmd *exec.Cmd) error {
	if e.ReturnStartErr != nil {
		return e.ReturnStartErr
	}

	if e.ReturnPID != 0 {
		cmd.Process = &os.Process{Pid: e.ReturnPID}
	}

	// mock output to stdout and stderr
	e.wg.Add(1)
	go e.writeToStdOut(cmd)
	e.wg.Add(1)
	go e.writeToStdErr(cmd)

	return nil
}

func (e *CmdExecutorMock) writeToStdOut(cmd *exec.Cmd) {
	defer e.wg.Done()

	for _, s := range e.ReturnStdOut {
		_, err := cmd.Stdout.Write([]byte(s + "\n"))
		if err != nil {
			log.Fatalf("Failed to write data into stdout: %s", err)
		}
	}
}

func (e *CmdExecutorMock) writeToStdErr(cmd *exec.Cmd) {
	defer e.wg.Done()

	for _, s := range e.ReturnStdErr {
		_, err := cmd.Stderr.Write([]byte(s + "\n"))
		if err != nil {
			log.Fatalf("Failed to write data into stderr: %s", err)
		}
	}
}

func (e *CmdExecutorMock) Wait(cmd *exec.Cmd) error {
	if e.ReturnWaitErr != nil {
		return e.ReturnWaitErr
	}
	e.wg.Wait()
	return nil
}
