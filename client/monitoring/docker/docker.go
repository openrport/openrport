package docker

import (
	"errors"
	"runtime"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share/logger"
)

var ErrorNotImplementedForOS = errors.New("docker support not implemented for " + runtime.GOOS)
var ErrorDockerNotAvailable = errors.New("docker executable not found on the system or the service is stopped")

type Handler struct {
	logger                            *chshare.Logger
	dockerIsAvailable                 bool
	dockerAvailabilityLastRequestedAt *time.Time
}

func NewHandler(logger *chshare.Logger) *Handler {
	return &Handler{logger: logger}
}
