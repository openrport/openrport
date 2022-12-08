package caddy

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

type Server struct {
	cfg    *Config
	logger *logger.Logger
	errCh  chan error
}

func CheckExecExists(path string, filesAPI files.FileAPI) (exists bool, err error) {
	exists, err = filesAPI.Exist(path)
	if err != nil {
		return false, err
	}

	return exists, nil

}

func GetServerVersion(ctx context.Context, cfg *Config) (majorVersion int, err error) {
	cmd := exec.CommandContext(ctx, cfg.ExecPath, "version") // #nosec G204

	var b bytes.Buffer
	cmd.Stdout = &b
	// cmd.Stderr = &b
	err = cmd.Run()
	if err != nil {
		return -1, err
	}

	out := b.Bytes()
	majorVersion = int(out[1])
	return majorVersion, nil
}

func NewCaddyServer(cfg *Config, l *logger.Logger, errCh chan error) (c *Server) {
	c = &Server{
		cfg:    cfg,
		logger: l,
		errCh:  errCh,
	}
	return c
}

func (c *Server) Start(ctx context.Context) {
	confFile := "caddy-base.conf"
	configParam := fmt.Sprintf("%s/%s", c.cfg.DataDir, confFile)

	c.logger.Debugf("caddy exec path: %s", c.cfg.ExecPath)
	c.logger.Debugf("caddy config: %s", configParam)

	cmd := exec.CommandContext(ctx, c.cfg.ExecPath, "run", "--config", configParam, "--adapter", "caddyfile") // #nosec G204
	defer func() {
		c.errCh <- errors.New("caddy server closed")
	}()

	r, w := io.Pipe()
	cmd.Stdout = w
	cmd.Stderr = w

	lines := make(chan string)

	go func() {
		for line := range lines {
			c.logger.Debugf("log: " + line)
		}
	}()

	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("scanner err = %+v\n", err)
		}
	}()

	c.logger.Debugf("server running")
	err := cmd.Run()
	if err != nil {
		if strings.Contains(err.Error(), "signal: killed") {
			c.logger.Infof("server: %v", err)
		} else {
			c.logger.Errorf("server error: %v", err)
		}

		if c.errCh != nil {
			c.errCh <- err
		}
	}
	c.logger.Debugf("server stopped")
}
