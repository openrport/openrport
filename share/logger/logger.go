package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
)

type LogLevel int

const (
	LogLevelError LogLevel = 0
	LogLevelInfo  LogLevel = 1
	LogLevelDebug LogLevel = 2
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelInfo:
		return "info"
	case LogLevelDebug:
		return "debug"
	case LogLevelError:
		return "error"
	default:
		return ""
	}
}

func ParseLogLevel(str string) (LogLevel, error) {
	var m = map[string]LogLevel{
		"error": LogLevelError,
		"info":  LogLevelInfo,
		"debug": LogLevelDebug,
	}
	if result, ok := m[str]; ok {
		return result, nil
	}
	return LogLevelError, fmt.Errorf("invalid log level: %q", str)
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
	Level  LogLevel
}

func NewLogger(prefix string, output LogOutput, level LogLevel) *Logger {
	l := &Logger{
		prefix: prefix,
		logger: log.New(output.File, "", log.Ldate|log.Ltime),
		output: output,
		Level:  level,
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
	if l.Level >= severity {
		l.logger.Printf(severity.String()+": "+l.prefix+": "+f, args...)
	}
}

func (l *Logger) Fork(prefix string, args ...interface{}) *Logger {
	// slip the parent prefix at the front
	args = append([]interface{}{l.prefix}, args...)
	ll := NewLogger(fmt.Sprintf("%s: "+prefix, args...), l.output, l.Level)
	return ll
}

func (l *Logger) Prefix() string {
	return l.prefix
}

type ILogger interface {
	Errorf(f string, args ...interface{})
	Infof(f string, args ...interface{})
	Debugf(f string, args ...interface{})
	Logf(severity LogLevel, f string, args ...interface{})
	Fork(prefix string, args ...interface{})
	Prefix() string
}

type LogControl map[string]bool

type LogController struct {
	control       LogControl
	defaultActive bool

	gate sync.RWMutex
}

var DefaultActiveDynamicLogs = true

func newLogController(defaultActive bool) (lc *LogController) {
	lc = &LogController{
		control:       make(LogControl),
		defaultActive: defaultActive,
	}
	return lc
}

func (lc *LogController) SetControl(name string, active bool) {
	lc.gate.Lock()
	defer lc.gate.Unlock()
	lc.control[name] = active
}

func (lc *LogController) IsActive(name string) (active bool) {
	lc.gate.RLock()
	defer lc.gate.RUnlock()
	active, isSet := lc.control[name]
	if !isSet && lc.defaultActive {
		return true
	}
	return lc.control[name]
}

func (lc *LogController) DefaultActive() (defaultActive bool) {
	lc.gate.RLock()
	defer lc.gate.RUnlock()
	return lc.defaultActive
}

func (lc *LogController) Clone() (clonedLC *LogController) {
	lc.gate.RLock()
	defer lc.gate.RUnlock()

	clonedLC = &LogController{}
	clonedLC.defaultActive = lc.defaultActive

	clonedLC.control = make(LogControl, len(lc.control))
	for name, status := range lc.control {
		clonedLC.control[name] = status
	}

	return clonedLC
}

type DynamicLogger struct {
	*Logger
	*LogController
}

func NewDynamicLogger(prefix string, output LogOutput, level LogLevel, enabled bool, defaultActive bool) (dl *DynamicLogger) {
	dl = &DynamicLogger{
		LogController: newLogController(defaultActive),
		Logger:        NewLogger(prefix, output, level),
	}
	dl.LogController.SetControl(prefix, enabled)
	return dl
}

func ForkToDynamicLogger(l *Logger, prefix string, enabled bool, defaultActive bool) (dl *DynamicLogger) {
	dl = &DynamicLogger{
		LogController: newLogController(defaultActive),
		Logger:        l.Fork(prefix),
	}
	dl.SetControl(prefix, enabled)
	return dl
}

func (d *DynamicLogger) GetLogger() (l *Logger) {
	return d.Logger
}

func (d *DynamicLogger) Fork(prefix string, args ...interface{}) (dl *DynamicLogger) {
	newPrefix := fmt.Sprintf(prefix, args...)
	// inherit active status and defaultActive from parent logger
	dl = ForkToDynamicLogger(d.Logger, newPrefix, d.IsActive(d.prefix), d.DefaultActive())
	// inherit named logs control from the parent dynamic logger
	dl.LogController = d.LogController.Clone()
	// remove the parent logger prefix from control
	dl.SetControl(d.Prefix(), false)
	// ensure the new logger is enabled
	dl.SetControl(dl.Logger.Prefix(), true)
	return dl
}

func (d *DynamicLogger) Errorf(f string, args ...interface{}) {
	d.Logf(LogLevelError, f, args...)
}

func (d *DynamicLogger) Infof(f string, args ...interface{}) {
	d.Logf(LogLevelInfo, f, args...)
}

func (d *DynamicLogger) Debugf(f string, args ...interface{}) {
	d.Logf(LogLevelDebug, f, args...)
}

func (d *DynamicLogger) Logf(severity LogLevel, f string, args ...interface{}) {
	d.NLogf("", severity, f, args...)
}

func (d *DynamicLogger) NErrorf(name string, f string, args ...interface{}) {
	d.NLogf(name, LogLevelError, f, args...)
}

func (d *DynamicLogger) NInfof(name string, f string, args ...interface{}) {
	d.NLogf(name, LogLevelInfo, f, args...)
}

func (d *DynamicLogger) NDebugf(name string, f string, args ...interface{}) {
	d.NLogf(name, LogLevelDebug, f, args...)
}

func (d *DynamicLogger) NLogf(name string, severity LogLevel, f string, args ...interface{}) {
	if name != "" && !d.LogController.IsActive(name) {
		return
	}
	if name == "" && !d.LogController.IsActive(d.prefix) {
		return
	}
	if d.Level >= severity {
		if name == "" {
			d.logger.Printf(severity.String()+": "+d.Prefix()+": "+f, args...)
		} else {
			d.logger.Printf(severity.String()+": "+d.Prefix()+": "+name+": "+f, args...)
		}
	}
}
