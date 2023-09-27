package transformers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/openrport/openrport/share/models"

	"github.com/openrport/openrport/plus/capabilities/alerting/entities/measures"
)

type RawMountPointsInfo map[string]uint64

type mountPointsList map[string]*measures.MountPoint

func TransformRportMeasurementToMeasure(rm *models.Measurement) (m *measures.Measure, err error) {
	m = &measures.Measure{}

	m.ClientID = rm.ClientID
	m.Timestamp = rm.Timestamp
	m.CPUUsagePercent = rm.CPUUsagePercent
	m.MemoryUsagePercent = rm.MemoryUsagePercent
	m.IoUsagePercent = rm.IoUsagePercent
	if rm.NetLan != nil {
		m.NetLan = *rm.NetLan
	}
	if rm.NetWan != nil {
		m.NetWan = *rm.NetWan
	}

	if rm.Processes != "" {
		pl, err := TransformProcessesJSONToProcesses(rm.Processes)
		if err != nil {
			return nil, err
		}
		m.Processes = pl
	}

	if rm.Mountpoints != "" {
		mp, err := TransformMountPointsJSONToMountPoints(rm.Mountpoints)
		if err != nil {
			return nil, err
		}
		m.MountPoints = mp
	}

	return m, nil
}

func TransformProcessesJSONToProcesses(ps string) (processList []measures.Process, err error) {
	processList = make([]measures.Process, 0)

	err = json.Unmarshal([]byte(ps), &processList)
	if err != nil {
		return nil, err
	}

	return processList, nil
}

func TransformMountPointsJSONToMountPoints(mpJSON string) (mountPoints []measures.MountPoint, err error) {
	mountPoints = make([]measures.MountPoint, 0, 1)
	mpInfo := RawMountPointsInfo{}

	// extract the array of rows containing the mount point vales
	err = json.Unmarshal([]byte(mpJSON), &mpInfo)
	if err != nil {
		return nil, err
	}

	// as we collect the details row by row we'll need to store the row results for a mount point
	mpl := make(mountPointsList, 0)

	// for each row of mount point info with corresponding value
	for key, value := range mpInfo {
		// get the mount point type (e.g. free_b or total_b) and the mount name
		parts := strings.Split(key, ".")
		if len(parts) != 2 {
			return nil, fmt.Errorf("unable to process mount point info item: %s", key)
		}
		valueType := parts[0]
		name := parts[1]

		// check if the mount point is already in the mount point details collected so far
		mp, ok := mpl[name]
		if !ok {
			// if we don't know the mount point name then add to the mount point list
			mp = &measures.MountPoint{}
			mp.Name = name
			mpl[name] = mp
		}

		// map the mount point value to the mount point amount field
		switch valueType {
		case "free_b":
			{
				mp.FreeBytes = value
			}
		case "total_b":
			{
				mp.TotalBytes = value
			}
		default:
			{
				return nil, fmt.Errorf("unable to process mount point value type: %s", key)
			}
		}
	}

	// copy the wip mount point list/map to an array
	for _, mps := range mpl {
		m := *mps
		m.FreePercent = mps.CalcFreePercent()
		m.UsedPercent = mps.CalcUsedPercent()
		mountPoints = append(mountPoints, m)
	}

	sort.Slice(mountPoints, func(i int, j int) bool {
		return mountPoints[i].Name < mountPoints[j].Name
	})

	return mountPoints, nil
}
