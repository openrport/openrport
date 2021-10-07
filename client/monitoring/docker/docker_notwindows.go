// +build !windows

package docker

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cloudradar-monitoring/rport/client/common"
)

const dockerAvailabilityCheckCacheExpiration = 1 * time.Minute
const cmdExecTimeout = 10 * time.Second

var dockerIsAvailable bool
var dockerAvailabilityLastRequestedAt *time.Time

// isDockerAvailable maintains a simple cache to prevent executing shell commands too often
func (h *Handler) isDockerAvailable() bool {
	now := time.Now()
	if dockerAvailabilityLastRequestedAt != nil &&
		now.Sub(*dockerAvailabilityLastRequestedAt) < dockerAvailabilityCheckCacheExpiration {
		return dockerIsAvailable
	}

	_, err := exec.LookPath("docker")
	dockerIsAvailable = err == nil

	if dockerIsAvailable {
		dockerPrefix := ""
		if runtime.GOOS == "linux" {
			dockerPrefix = "sudo "
		}

		_, err := common.RunCommandWithTimeout(cmdExecTimeout, "/bin/sh", "-c", dockerPrefix+"docker info")
		if err != nil {
			h.logger.Debugf("while executing 'docker info' to check if docker is available:%v", err)
		}
		dockerIsAvailable = dockerIsAvailable && (err == nil)
	}

	dockerAvailabilityLastRequestedAt = &now
	return dockerIsAvailable
}

// ContainerNameByID returns the name of a container identified by its id
func (h *Handler) ContainerNameByID(id string) (string, error) {
	if !h.isDockerAvailable() {
		return "", ErrorDockerNotAvailable
	}

	out, err := common.RunCommandWithTimeout(cmdExecTimeout, "/bin/sh", "-c", fmt.Sprintf("sudo docker inspect --format \"{{ .Name }}\" %s", id))
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = errors.New(ee.Error() + ": " + string(ee.Stderr))
		}

		return "", err
	}

	// remove \n and possible spaces around
	name := strings.TrimSpace(string(out))

	// remove leading slash from the name
	name = strings.TrimPrefix(name, "/")

	return name, nil
}
