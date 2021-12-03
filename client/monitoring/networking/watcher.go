package networking

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	utilnet "github.com/shirou/gopsutil/net"
	"github.com/sirupsen/logrus"

	"github.com/cloudradar-monitoring/rport/client/monitoring/helper"
)

type NetWatcherConfig struct {
	NetInterfaceExclude             []string
	NetInterfaceExcludeRegex        []string
	NetInterfaceExcludeDisconnected bool
	NetInterfaceExcludeLoopback     bool
	NetMetrics                      []string
	NetInterfaceMaxSpeed            uint64
}

type NetWatcher struct {
	config NetWatcherConfig

	lastIOCounters   []utilnet.IOCountersStat
	lastIOCountersAt *time.Time

	netInterfaceExcludeRegexCompiled []*regexp.Regexp
	constantlyExcludedInterfaceCache map[string]bool
}

func NewWatcher(cfg NetWatcherConfig) *NetWatcher {
	return &NetWatcher{
		config:                           cfg,
		constantlyExcludedInterfaceCache: map[string]bool{},
	}
}

// InterfaceExcludeRegexCompiled compiles and cache all the interfaces-filtering regexp's user has specified in the Config
// So we don't need to compile them on each iteration of measurements
func (nw *NetWatcher) InterfaceExcludeRegexCompiled() []*regexp.Regexp {
	if len(nw.netInterfaceExcludeRegexCompiled) > 0 {
		return nw.netInterfaceExcludeRegexCompiled
	}

	if len(nw.config.NetInterfaceExcludeRegex) > 0 {
		for _, reString := range nw.config.NetInterfaceExcludeRegex {
			re, err := regexp.Compile(reString)

			if err != nil {
				logrus.Errorf("[NET] net_interface_exclude_regex regexp '%s' compile error: %s", reString, err.Error())
				continue
			}
			nw.netInterfaceExcludeRegexCompiled = append(nw.netInterfaceExcludeRegexCompiled, re)
		}
	}

	return nw.netInterfaceExcludeRegexCompiled
}

func (nw *NetWatcher) isInterfaceExcludedByName(netIf *utilnet.InterfaceStat) bool {
	for _, excludedIf := range nw.config.NetInterfaceExclude {
		if strings.EqualFold(netIf.Name, excludedIf) {
			return true
		}
	}
	return false
}

func (nw *NetWatcher) isInterfaceExcludedByRegexp(netIf *utilnet.InterfaceStat) bool {
	for _, re := range nw.InterfaceExcludeRegexCompiled() {
		if re.MatchString(netIf.Name) {
			return true
		}
	}
	return false
}

func (nw *NetWatcher) ExcludedInterfacesByName(allInterfaces []utilnet.InterfaceStat) map[string]struct{} {
	excludedInterfaces := map[string]struct{}{}

	for _, netIf := range allInterfaces {
		// use a cache for excluded interfaces, because all the checks(except UP/DOWN state) are constant for the same interface&Config
		if isExcluded, cacheExists := nw.constantlyExcludedInterfaceCache[netIf.Name]; cacheExists {
			if isExcluded ||
				nw.config.NetInterfaceExcludeDisconnected && isInterfaceDown(&netIf) {
				// interface is found excluded in the cache or has a DOWN state
				excludedInterfaces[netIf.Name] = struct{}{}
				logrus.Debugf("[NET] interface excluded: %s", netIf.Name)
				continue
			}
		} else {
			if nw.config.NetInterfaceExcludeLoopback && isInterfaceLoobpack(&netIf) ||
				nw.isInterfaceExcludedByName(&netIf) ||
				nw.isInterfaceExcludedByRegexp(&netIf) {
				// add the excluded interface to the cache because this checks are constant
				nw.constantlyExcludedInterfaceCache[netIf.Name] = true
			} else if nw.config.NetInterfaceExcludeDisconnected && isInterfaceDown(&netIf) {
				// exclude DOWN interface for now
				// lets cache it as false and then we will only check UP/DOWN status
				nw.constantlyExcludedInterfaceCache[netIf.Name] = false
			} else {
				// interface is not excluded
				nw.constantlyExcludedInterfaceCache[netIf.Name] = false
				continue
			}

			excludedInterfaces[netIf.Name] = struct{}{}
			logrus.Debugf("[NET] interface excluded: %s", netIf.Name)
		}
	}
	return excludedInterfaces
}

// fillEmptyMeasurements used to fill measurements with nil's for all non-excluded interfaces
// It is called in case measurements are not yet ready or some error happens while retrieving counters
func (nw *NetWatcher) fillEmptyMeasurements(results helper.MeasurementsMap, interfaces []utilnet.InterfaceStat, excludedInterfacesByName map[string]struct{}) {
	for _, netIf := range interfaces {
		if _, isExcluded := excludedInterfacesByName[netIf.Name]; isExcluded {
			continue
		}

		for _, metric := range nw.config.NetMetrics {
			if strings.HasPrefix(metric, "total") {
				continue
			}
			results[metric+"."+netIf.Name] = nil
		}
	}
}

// fillCountersMeasurements used to fill measurements with nil's for all non-excluded interfaces
func (nw *NetWatcher) fillCountersMeasurements(results helper.MeasurementsMap, interfaces []utilnet.InterfaceStat, excludedInterfacesByName map[string]struct{}) error {
	counters, err := getNetworkIOCounters()
	if err != nil {
		// fill empty measurements for not-excluded interfaces
		nw.fillEmptyMeasurements(results, interfaces, excludedInterfacesByName)
		return fmt.Errorf("Failed to read IOCounters: %s", err.Error())
	}

	gotIOCountersAt := time.Now()
	defer func() {
		nw.lastIOCountersAt = &gotIOCountersAt
		nw.lastIOCounters = counters
	}()

	if nw.lastIOCounters == nil {
		logrus.Debugf("[NET] IO stat is available starting from 2nd check")
		nw.fillEmptyMeasurements(results, interfaces, excludedInterfacesByName)
		// do not need to return the error here, because this is normal behavior
		return nil
	}

	lastIOCounterByName := map[string]utilnet.IOCountersStat{}
	for _, lastIOCounter := range nw.lastIOCounters {
		lastIOCounterByName[lastIOCounter.Name] = lastIOCounter
	}

	secondsSinceLastMeasurement := gotIOCountersAt.Sub(*nw.lastIOCountersAt).Seconds()

	var totalBytesReceivedPerSecond uint64
	var totalBytesSentPerSecond uint64

	linkSpeedProvider := newLinkSpeedProvider()
	for _, ioCounter := range counters {
		// iterate over all counters
		// each ioCounter corresponds to the specific interface with name ioCounter.Name
		if _, isExcluded := excludedInterfacesByName[ioCounter.Name]; isExcluded {
			continue
		}

		var previousIOCounter utilnet.IOCountersStat
		var exists bool
		// found prev counter data
		if previousIOCounter, exists = lastIOCounterByName[ioCounter.Name]; !exists {
			logrus.Errorf("[NET] Previous IOCounters stat not found: %s", ioCounter.Name)
			continue
		}

		for _, metric := range nw.config.NetMetrics {
			switch metric {
			case "in_B_per_s":
				bytesReceivedSinceLastMeasurement := ioCounter.BytesRecv - previousIOCounter.BytesRecv
				totalBytesReceivedPerSecond += bytesReceivedSinceLastMeasurement
				results[metric+"."+ioCounter.Name] = helper.FloatToIntRoundUP(float64(bytesReceivedSinceLastMeasurement) / secondsSinceLastMeasurement)
			case "out_B_per_s":
				bytesSentSinceLastMeasurement := ioCounter.BytesSent - previousIOCounter.BytesSent
				totalBytesSentPerSecond += bytesSentSinceLastMeasurement
				results[metric+"."+ioCounter.Name] = helper.FloatToIntRoundUP(float64(bytesSentSinceLastMeasurement) / secondsSinceLastMeasurement)
			case "errors_per_s":
				errorsSinceLastMeasurement := ioCounter.Errin + ioCounter.Errout - previousIOCounter.Errin - previousIOCounter.Errout
				results[metric+"."+ioCounter.Name] = helper.FloatToIntRoundUP(float64(errorsSinceLastMeasurement) / secondsSinceLastMeasurement)
			case "dropped_per_s":
				droppedSinceLastMeasurement := ioCounter.Dropin + ioCounter.Dropout - ioCounter.Dropin - previousIOCounter.Dropout
				results[metric+"."+ioCounter.Name] = helper.FloatToIntRoundUP(float64(droppedSinceLastMeasurement) / secondsSinceLastMeasurement)
			}
		}

		currLinkSpeed := float64(ioCounter.BytesRecv-previousIOCounter.BytesRecv+ioCounter.BytesSent-previousIOCounter.BytesSent) / secondsSinceLastMeasurement

		maxAvailableLinkSpeed := float64(nw.config.NetInterfaceMaxSpeed)
		if maxAvailableLinkSpeed == 0 {
			maxAvailableLinkSpeed, err = linkSpeedProvider.GetMaxAvailableLinkSpeed(ioCounter.Name)
			if err != nil {
				logrus.WithError(err).Debugf("[NET] cannot get max available link speed for %s. Skipping net_util_percent metric...", ioCounter.Name)
				continue
			}
		}

		if maxAvailableLinkSpeed < currLinkSpeed {
			// some network adapters can report incorrect link speed value
			maxAvailableLinkSpeed = currLinkSpeed
		}

		results["net_util_percent."+ioCounter.Name] = helper.RoundToTwoDecimalPlaces((currLinkSpeed / maxAvailableLinkSpeed) * 100)
	}

	for _, metric := range nw.config.NetMetrics {
		switch metric {
		case "total_in_B_per_s":
			results[metric] = helper.FloatToIntRoundUP(float64(totalBytesReceivedPerSecond) / secondsSinceLastMeasurement)
		case "total_out_B_per_s":
			results[metric] = helper.FloatToIntRoundUP(float64(totalBytesSentPerSecond) / secondsSinceLastMeasurement)
		}
	}
	return nil
}

func (nw *NetWatcher) Results() (helper.MeasurementsMap, error) {
	results := helper.MeasurementsMap{}

	interfaces, err := utilnet.Interfaces()
	if err != nil {
		logrus.Errorf("[NET] Failed to read interfaces: %s", err.Error())
		return nil, err
	}

	excludedInterfacesByNameMap := nw.ExcludedInterfacesByName(interfaces)
	// fill counters measurements into results
	err = nw.fillCountersMeasurements(results, interfaces, excludedInterfacesByNameMap)
	if err != nil {
		logrus.Errorf("[NET] Failed to collect counters: %s", err.Error())
		return results, err
	}

	return results, nil
}
