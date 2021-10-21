package middleware

import (
	"fmt"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type RecoveryLogger struct {
	*chshare.Logger
}

func NewRecoveryLogger(l *chshare.Logger) *RecoveryLogger {
	return &RecoveryLogger{
		Logger: l,
	}
}

func (l *RecoveryLogger) Println(v ...interface{}) {
	l.Errorf(fmt.Sprintln(v...))
}
