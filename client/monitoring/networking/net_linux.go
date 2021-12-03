// +build linux

package networking

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type linuxLinkSpeedProvider struct {
}

func newLinkSpeedProvider() linkSpeedProvider {
	return &linuxLinkSpeedProvider{}
}

func (p *linuxLinkSpeedProvider) GetMaxAvailableLinkSpeed(ifName string) (float64, error) {
	data, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/speed", ifName))
	if err != nil {
		return 0, errors.Wrap(err, "cannot read speed info file")
	}

	strData := strings.TrimSpace(string(data))
	megaBitsPerSecond, err := strconv.Atoi(strData)
	if err != nil {
		return 0, errors.Wrap(err, "invalid data in speed info file")
	}

	if megaBitsPerSecond < 0 {
		return 0, fmt.Errorf("got unexpected speed value: %s", strData)
	}

	return float64(megaBitsPerSecond) / 8 * 1000 * 1000, nil
}
