// +build !windows,!linux

package networking

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type ifconfigLinkSpeedProvider struct {
}

func newLinkSpeedProvider() linkSpeedProvider {
	return &ifconfigLinkSpeedProvider{}
}

func (p *ifconfigLinkSpeedProvider) GetMaxAvailableLinkSpeed(ifName string) (float64, error) {
	cmd := exec.Command("ifconfig", "-v", ifName)

	var outb bytes.Buffer
	cmd.Stdout = &outb
	err := cmd.Run()
	if err != nil {
		return 0, errors.Wrap(err, "ifconfig failed")
	}

	scanner := bufio.NewScanner(&outb)
	for scanner.Scan() {
		lineParts := strings.Fields(scanner.Text())
		if len(lineParts) < 4 {
			continue
		}

		if lineParts[0] == "link" && lineParts[1] == "rate:" {
			value, err := strconv.ParseFloat(lineParts[2], 64)
			if err != nil {
				return 0, errors.Wrap(err, "cannot parse link rate value")
			}

			unitsStr := lineParts[3]
			return calcBytesPerSecond(value, unitsStr)
		}
	}
	return 0, errors.New("wasn't able to find link rate info")
}

func calcBytesPerSecond(value float64, units string) (float64, error) {
	valueInBytes := value / 8
	switch units {
	case "Gbps":
		return valueInBytes * 1000 * 1000 * 1000, nil
	case "Mbps":
		return valueInBytes * 1000 * 1000, nil
	case "Kbps":
		return valueInBytes * 1000, nil
	case "bps":
		return valueInBytes, nil
	}
	return 0, fmt.Errorf("unsupported unit of measure: %s", units)
}
