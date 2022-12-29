package caddy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

type Server struct {
	cmd        *exec.Cmd
	ctx        context.Context
	cfg        *Config
	logger     *logger.Logger
	errCh      chan error
	httpClient http.Client

	// for forwarding caddy logs into the rportd log
	logLogger *logger.Logger

	// readers below used in a pipe that we use for relaying caddy log messages into the rport log
	w     *io.PipeWriter // where caddy should write log messages
	r     *io.PipeReader // where rport should read caddy log messages
	lines chan string    // we'll write these lines to the rportd log
}

// caddy version string -> v2.6.2 h1:wKoFIxpmOJLGl3QXoo6PNbYvGW4xLEgo32GPBEjWL8o=
var versionStringMatch = regexp.MustCompile(`^v(.*)\..*\..*.* `)

func ExtractCaddyMajorVersionStr(versionInfo string) (majorVersion string) {
	matches := versionStringMatch.FindAllStringSubmatch(versionInfo, -1)
	if len(matches) > 0 && len(matches[0]) > 0 {
		majorVersion = matches[0][1]
	}
	return
}

func GetExecVersion(cfg *Config) (majorVersion int, err error) {
	cmd := exec.Command(cfg.ExecPath, "version") // #nosec G204

	var b bytes.Buffer
	cmd.Stdout = &b
	err = cmd.Run()
	if err != nil {
		return -1, err
	}

	// caddy version output
	out := b.String()

	majorVersionStr := ExtractCaddyMajorVersionStr(out)

	majorVersion, err = strconv.Atoi(majorVersionStr)
	if err != nil {
		return -1, errors.New("unable to process caddy version")
	}

	return majorVersion, nil
}

func NewCaddyServer(cfg *Config, l *logger.Logger) (c *Server) {
	errCh := make(chan error)
	httpClient := newHTTPDomainSocketClient()

	c = &Server{
		httpClient: httpClient,
		cfg:        cfg,
		logger:     l,
		logLogger:  l.Fork("log"),
		errCh:      errCh,
	}
	return c
}

func (c *Server) Start(ctx context.Context) (err error) {
	configParam := c.cfg.MakeBaseConfFilename()

	c.logger.Debugf("caddy exec path: %s", c.cfg.ExecPath)
	c.logger.Debugf("caddy config: %s", configParam)

	c.cmd = exec.CommandContext(ctx, c.cfg.ExecPath, "run", "--config", configParam, "--adapter", "caddyfile") // #nosec G204
	c.ctx = ctx

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
	select {
	case <-c.ctx.Done():
		err = c.ctx.Err()
	case err = <-c.errCh:
	}

	c.logger.Debugf("%v", err)

	return err
}

func (c *Server) Close() (err error) {
	// close the standard io pipes
	err = c.r.Close()
	if err != nil {
		c.logger.Infof("error closing caddy log reader: %v", err)
	}
	err = c.w.Close()
	if err != nil {
		c.logger.Infof("error closing caddy log writer: %v, err")
	}

	c.logger.Debugf("stopped")
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
		level, err := extractCaddyLogLevel(line)
		// err indicates wasn't a caddy log message. we don't really care what the err is.
		if err != nil {
			c.logger.Debugf(line)
			continue
		}

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

type logMessage struct {
	Level string `json:"level"`
}

func extractCaddyLogLevel(line string) (level string, err error) {
	logLine := &logMessage{}
	err = json.Unmarshal([]byte(line), logLine)
	if err != nil {
		return "debug", err
	}

	return logLine.Level, nil
}
