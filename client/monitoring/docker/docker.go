package docker

import (
	"errors"
	"runtime"
	"time"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

var ErrorNotImplementedForOS = errors.New("docker support not implemented for " + runtime.GOOS)
var ErrorDockerNotAvailable = errors.New("docker executable not found on the system or the service is stopped")

type Handler struct {
	logger                            *logger.Logger
	dockerIsAvailable                 bool
	dockerAvailabilityLastRequestedAt *time.Time
}

func NewHandler(logger *logger.Logger) *Handler {
	return &Handler{logger: logger}
}
