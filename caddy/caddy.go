package caddy

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

type Server struct {
	cmd       *exec.Cmd
	lines     chan string
	r         *io.PipeReader
	w         *io.PipeWriter
	cfg       *Config
	logger    *logger.Logger
	logLogger *logger.Logger
	errCh     chan error
}

func ExecExists(path string, filesAPI files.FileAPI) (exists bool, err error) {
	exists, err = filesAPI.Exist(path)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func GetExecVersion(cfg *Config) (majorVersion int, err error) {
	ctx := context.Background()
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
		cfg:       cfg,
		logger:    l,
		logLogger: l.Fork("log"),
		errCh:     errCh,
	}
	return c
}

func (c *Server) Start(ctx context.Context) (err error) {
	configParam := c.cfg.MakeBaseConfFilename()

	c.logger.Debugf("caddy exec path: %s", c.cfg.ExecPath)
	c.logger.Debugf("caddy config: %s", configParam)

	c.cmd = exec.CommandContext(ctx, c.cfg.ExecPath, "run", "--config", configParam, "--adapter", "caddyfile") // #nosec G204

	c.r, c.w = io.Pipe()
	c.cmd.Stdout = c.w
	c.cmd.Stderr = c.w

	c.lines = make(chan string)

	go c.readCaddyOutputLines()
	go c.writeCaddyOutputToLog()

	c.logger.Debugf("server starting")

	err = c.cmd.Start()
	if err != nil {
		c.logger.Errorf("caddy server failed to start, reason: %s", err)
		return err
	}

	c.Run()
	return nil
}

func (c *Server) Run() {
	go func() {
		c.logger.Debugf("running")
		err := c.cmd.Wait()
		if err != nil {
			c.logger.Errorf("caddy server stopping, reason: %s", err)
			c.errCh <- err
		}
	}()
}

func (c *Server) Wait() (err error) {
	c.logger.Debugf("watching for errors")
	err = <-c.errCh

	if strings.Contains(err.Error(), "signal: killed") {
		// valid shutdown
		c.logger.Debugf("server: %v", err)
	} else {
		// caddy not happy so quit rportd
		log.Fatalf("caddy server error: %v", err)
	}

	return err
}

func (c *Server) Close() (err error) {
	c.logger.Debugf("close requested")
	// close the standard io pipes
	c.r.Close()
	c.w.Close()
	return nil
}

func (c *Server) readCaddyOutputLines() {
	defer close(c.lines)
	scanner := bufio.NewScanner(c.r)
	for scanner.Scan() {
		c.lines <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		c.logger.Debugf("scanner err: %+v\n", err)
	}
}

var caddyToRportLogLevelMap = map[string]string{
	"debug": "debug",
	"info":  "info",
	"warn":  "info",
	"error": "error",
	"fatal": "error",
	"panic": "error",
}

func (c *Server) writeCaddyOutputToLog() {
	for line := range c.lines {
		level := extractCaddyLogLevel(line)
		mapsTo, ok := caddyToRportLogLevelMap[level]
		if ok {
			switch mapsTo {
			case "debug":
				c.logLogger.Debugf(line)
			case "info":
				c.logLogger.Infof(line)
			case "error":
				c.logLogger.Errorf(line)
			}
		} else {
			c.logLogger.Debugf(line)
		}
	}
}

var logLevelMatch = regexp.MustCompile("^{\"level\":\"(.*)\",\"ts\"")

func extractCaddyLogLevel(line string) (level string) {
	matches := logLevelMatch.FindAllStringSubmatch(line, -1)
	if len(matches) > 0 && len(matches[0]) > 0 {
		level = matches[0][1]
	}
	return level
}
