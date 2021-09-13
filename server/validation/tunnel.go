package validation

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

const idleTimeoutDefault = time.Minute * 5
const idleTimeoutMax = time.Hour * 24 * 7 //a week
const idleTimeoutMin = time.Duration(0)

func ResolveIdleTunnelTimeoutValue(idleTimeoutMinutesStr string, skipIdleTimeout bool) (time.Duration, error) {
	if idleTimeoutMinutesStr != "" && skipIdleTimeout {
		return 0, errors2.APIError{
			Message: fmt.Sprintf(
				"conflicting parameters idle timeout %s and skip idle timeout %s, either decide for skipping an idle timeout or provide a non empty timeout value",
				idleTimeoutMinutesStr,
				idleTimeoutMinutesStr,
			),
			HTTPStatus: http.StatusBadRequest,
		}
	}
	if skipIdleTimeout {
		return 0, nil
	}

	if idleTimeoutMinutesStr == "" {
		return idleTimeoutDefault, nil
	}

	idleTimeoutMinutesInt, err := strconv.Atoi(idleTimeoutMinutesStr)
	if err != nil {
		return 0, errors2.APIError{
			Message:    "invalid idle timeout param",
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		}
	}
	idleTimeoutMinutes := time.Duration(idleTimeoutMinutesInt) * time.Minute

	if idleTimeoutMin > idleTimeoutMinutes || idleTimeoutMinutes > idleTimeoutMax {
		return 0, errors2.APIError{
			Message:    fmt.Sprintf("idle timeoout param should be in range [%d,%d]", idleTimeoutMin, idleTimeoutMax),
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		}
	}

	return idleTimeoutMinutes, nil
}
