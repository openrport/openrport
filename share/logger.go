package chshare

import (
	"fmt"
	"log"
	"os"
)

type LogLevel int

const (
	LogLevelError LogLevel = 0
	LogLevelInfo  LogLevel = 1
	LogLevelDebug LogLevel = 2
)

func ParseLogLevel(str string) (LogLevel, error) {
	var m = map[string]LogLevel{
		"error": LogLevelError,
		"info":  LogLevelInfo,
		"debug": LogLevelDebug,
	}
	if result, ok := m[str]; ok {
		return result, nil
	}
	return LogLevelError, fmt.Errorf("invalid log level")
}

type LogOutput struct {
	File     *os.File
	filePath string
}

func NewLogOutput(filePath string) LogOutput {
	return LogOutput{
		filePath: filePath,
	}
}

func (o *LogOutput) Start() error {
	if o.filePath == "" {
		o.File = os.Stdout
		return nil
	}

	var err error
	o.File, err = os.OpenFile(o.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("can't open log file %s: %s", o.filePath, err)
	}
	return nil
}

func (o *LogOutput) Shutdown() {
	if o.File != nil && o.File != os.Stdout {
		_ = o.File.Close()
	}
}

type Logger struct {
	prefix string
	logger *log.Logger
	output LogOutput
	level  LogLevel
}

func NewLogger(prefix string, output LogOutput, level LogLevel) *Logger {
	l := &Logger{
		prefix: prefix,
		logger: log.New(output.File, "", log.Ldate|log.Ltime),
		output: output,
		level:  level,
	}
	return l
}

func (l *Logger) Errorf(f string, args ...interface{}) {
	l.Logf(LogLevelError, f, args...)
}

func (l *Logger) Infof(f string, args ...interface{}) {
	l.Logf(LogLevelInfo, f, args...)
}

func (l *Logger) Debugf(f string, args ...interface{}) {
	l.Logf(LogLevelDebug, f, args...)
}

func (l *Logger) Logf(severity LogLevel, f string, args ...interface{}) {
	if l.level >= severity {
		l.logger.Printf(l.prefix+": "+f, args...)
	}
}

func (l *Logger) FormatError(f string, args ...interface{}) error {
	return fmt.Errorf(l.prefix+": "+f, args...)
}

func (l *Logger) Fork(prefix string, args ...interface{}) *Logger {
	//slip the parent prefix at the front
	args = append([]interface{}{l.prefix}, args...)
	ll := NewLogger(fmt.Sprintf("%s: "+prefix, args...), l.output, l.level)
	return ll
}

func (l *Logger) Prefix() string {
	return l.prefix
}
