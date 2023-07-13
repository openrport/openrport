package measures

import (
	"reflect"
	"testing"
	"time"

	"github.com/realvnc-labs/rport/share/models"
)

func TestShouldCloneProcess(t *testing.T) {
	// Create sample data
	process := &Process{
		Name:    "Process 1",
		CmdLine: "Command 1",
	}

	clonedProcess := process.Clone()

	if process == &clonedProcess {
		t.Errorf("Clone should return a new object, but the reference is the same")
	}

	if !reflect.DeepEqual(process, &clonedProcess) {
		t.Errorf("Cloned process does not match the original process")
	}
}

func TestShouldCloneMountPoint(t *testing.T) {
	// Create sample data
	mountPoint := &MountPoint{
		Name:       "Mount Point 1",
		FreeBytes:  100,
		TotalBytes: 200,
	}

	clonedMountPoint := mountPoint.Clone()

	if mountPoint == &clonedMountPoint {
		t.Errorf("Clone should return a new object, but the reference is the same")
	}

	if !reflect.DeepEqual(mountPoint, &clonedMountPoint) {
		t.Errorf("Cloned mount point does not match the original mount point")
	}
}

func TestShouldCloneMeasure(t *testing.T) {
	// Create sample data
	processes := []Process{{Name: "Process 1", CmdLine: "Command 1"}}
	mountPoints := []MountPoint{{Name: "Mount Point 1", FreeBytes: 100, TotalBytes: 200}}
	measure := &Measure{
		UID:                "123",
		ClientID:           "456",
		Timestamp:          time.Now(),
		CPUUsagePercent:    50.5,
		MemoryUsagePercent: 60.5,
		IoUsagePercent:     70.5,
		NetLan:             models.NetBytes{In: 10, Out: 20},
		NetWan:             models.NetBytes{In: 30, Out: 40},
		Processes:          processes,
		MountPoints:        mountPoints,
	}

	clonedMeasure := measure.Clone()

	if measure == &clonedMeasure {
		t.Errorf("Clone should return a new object, but the reference is the same")
	}

	if !reflect.DeepEqual(measure, &clonedMeasure) {
		t.Errorf("Cloned measure does not match the original measure")
	}

	if !reflect.DeepEqual(measure.Processes, clonedMeasure.Processes) {
		t.Errorf("Cloned Processes slice does not match the original Processes slice")
	}

	if !reflect.DeepEqual(measure.MountPoints, clonedMeasure.MountPoints) {
		t.Errorf("Cloned MountPoints slice does not match the original MountPoints slice")
	}

	for i := range measure.Processes {
		if &measure.Processes[i] == &clonedMeasure.Processes[i] {
			t.Errorf("Cloned process at index %d is the same object as the original process", i)
		}
	}

	for i := range measure.MountPoints {
		if &measure.MountPoints[i] == &clonedMeasure.MountPoints[i] {
			t.Errorf("Cloned mount point at index %d is the same object as the original mount point", i)
		}
	}
}

func TestShouldCloneMeasures(t *testing.T) {
	// Create sample data
	netBytes := models.NetBytes{In: 10, Out: 20}
	processes := []Process{{Name: "Process 1", CmdLine: "Command 1"}}
	mountPoints := []MountPoint{{Name: "Mount Point 1", FreeBytes: 100, TotalBytes: 200}}
	measure := &Measure{
		UID:                "123",
		ClientID:           "456",
		Timestamp:          time.Now(),
		CPUUsagePercent:    50.5,
		MemoryUsagePercent: 60.5,
		IoUsagePercent:     70.5,
		NetLan:             netBytes,
		NetWan:             netBytes,
		Processes:          processes,
		MountPoints:        mountPoints,
	}
	measures := Measures{measure}

	clonedMeasures := measures.Clone()

	if &measures == &clonedMeasures {
		t.Errorf("Clone should return a new object, but the reference is the same")
	}

	if len(clonedMeasures) != len(measures) {
		t.Errorf("Expected length of clonedMeasures to be %d, got %d", len(measures), len(clonedMeasures))
	}

	for i := range measures {
		originalMeasure := measures[i]
		clonedMeasure := clonedMeasures[i]

		if originalMeasure == clonedMeasure {
			t.Errorf("Cloned measure at index %d is the same object as the original measure", i)
		}

		if !reflect.DeepEqual(originalMeasure, clonedMeasure) {
			t.Errorf("Cloned measure at index %d does not match the original measure", i)
		}

		if !reflect.DeepEqual(originalMeasure.Processes, clonedMeasure.Processes) {
			t.Errorf("Cloned Processes slice at index %d does not match the original Processes slice", i)
		}

		if !reflect.DeepEqual(originalMeasure.MountPoints, clonedMeasure.MountPoints) {
			t.Errorf("Cloned MountPoints slice at index %d does not match the original MountPoints slice", i)
		}

		for j := range originalMeasure.Processes {
			if &originalMeasure.Processes[j] == &clonedMeasure.Processes[j] {
				t.Errorf("Cloned process at index %d in measure %d is the same object as the original process", j, i)
			}
		}

		for j := range originalMeasure.MountPoints {
			if &originalMeasure.MountPoints[j] == &clonedMeasure.MountPoints[j] {
				t.Errorf("Cloned mount point at index %d in measure %d is the same object as the original mount point", j, i)
			}
		}
	}
}
