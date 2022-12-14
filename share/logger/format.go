package logger

import (
	"fmt"
	"time"
)

func FormatConnectionState(disconnectedAt *time.Time) string {
	if disconnectedAt != nil {
		return fmt.Sprintf("disconnected since %s", disconnectedAt)
	}
	return "connected"
}
