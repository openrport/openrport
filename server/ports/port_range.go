package ports

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
)

func TryParsePortRanges(portRanges []string) (mapset.Set, error) {
	result := mapset.NewSet()
	for _, rangeStr := range portRanges {
		rangeParts := strings.Split(rangeStr, "-")
		if len(rangeParts) == 1 {
			p, err := tryParsePortNumber(rangeParts[0])
			if err != nil {
				return nil, err
			}
			result.Add(p)
		} else if len(rangeParts) == 2 {
			portRange, err := tryParsePortNumberRange(rangeParts[0], rangeParts[1])
			if err != nil {
				return nil, err
			}
			result = result.Union(portRange)
		} else {
			return nil, fmt.Errorf("can't parse port range: incorrect range %s", rangeStr)
		}
	}
	return result, nil
}

func tryParsePortNumber(portNumberStr string) (int, error) {
	num, err := strconv.Atoi(portNumberStr)
	if err != nil {
		return 0, fmt.Errorf("can't parse port number %s: %s", portNumberStr, err)
	}
	if num < 0 || num > math.MaxUint16 {
		return 0, fmt.Errorf("invalid port number: %d", num)
	}
	return num, nil
}

func tryParsePortNumberRange(rangeStart, rangeEnd string) (mapset.Set, error) {
	start, err := tryParsePortNumber(rangeStart)
	if err != nil {
		return nil, err
	}
	end, err := tryParsePortNumber(rangeEnd)
	if err != nil {
		return nil, err
	}
	if start > end {
		return nil, fmt.Errorf("invalid port range %s-%s", rangeStart, rangeEnd)
	}
	return setFromRange(start, end), nil
}

func setFromRange(start, end int) mapset.Set {
	s := mapset.NewSet()
	for i := 0; i <= end-start; i++ {
		s.Add(start + i)
	}
	return s
}
