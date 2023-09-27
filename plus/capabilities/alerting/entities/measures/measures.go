package measures

import (
	"time"

	"github.com/openrport/openrport/share/models"
)

type Measure struct {
	UID                string          `json:"uid"` // unique id for idempotency
	ClientID           string          `json:"client_id"`
	Timestamp          time.Time       `json:"timestamp"`
	CPUUsagePercent    float64         `json:"cpu_usage_percent"`
	MemoryUsagePercent float64         `json:"memory_usage_percent"`
	IoUsagePercent     float64         `json:"io_usage_percent"`
	NetLan             models.NetBytes `json:"netlan"`
	NetWan             models.NetBytes `json:"netwan"`

	Processes   []Process    `json:"processes"`
	MountPoints []MountPoint `json:"mountpoints"`
}

type NetBytes struct {
	In  int `json:"in"`
	Out int `json:"out"`
}

type Process struct {
	Name    string `json:"name"`
	CmdLine string `json:"cmdline"`
}

type Measures []*Measure

func (ms Measures) Clone() (clonedMeasures Measures) {
	clonedMeasures = make(Measures, 0, len(ms))
	for _, m := range ms {
		clonedMeasure := m.Clone()
		clonedMeasures = append(clonedMeasures, &clonedMeasure)
	}
	return clonedMeasures
}

func (m *Measure) Clone() (clonedMeasure Measure) {
	clonedMeasure = *m
	clonedMeasure.Processes = []Process{}
	for _, process := range m.Processes {
		clonedMeasure.Processes = append(clonedMeasure.Processes, process.Clone())
	}
	clonedMeasure.MountPoints = []MountPoint{}
	for _, mp := range m.MountPoints {
		clonedMeasure.MountPoints = append(clonedMeasure.MountPoints, mp.Clone())
	}
	return clonedMeasure
}

func (p *Process) Clone() (clonedProcess Process) {
	clonedProcess = *p
	return clonedProcess
}

func (mp *MountPoint) Clone() (clonedMP MountPoint) {
	clonedMP = *mp
	return clonedMP
}
