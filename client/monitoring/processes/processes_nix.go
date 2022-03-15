//go:build !windows
// +build !windows

package processes

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/process"

	"github.com/cloudradar-monitoring/rport/client/monitoring/docker"
	"github.com/cloudradar-monitoring/rport/client/monitoring/helper"
)

var errorProcessTerminated = fmt.Errorf("Process was terminated")
var dockerContainerIDRE = regexp.MustCompile(`(?m)/docker/([a-f0-9]*)$`)

type ProcessCache struct {
	monitoredProcessCache map[int]*process.Process
}

type procStatus struct {
	PPID  int
	State string
}

func NewProcessCache() *ProcessCache {
	return &ProcessCache{monitoredProcessCache: map[int]*process.Process{}}
}

func (ph *ProcessHandler) processes(systemMemorySize uint64) ([]*ProcStat, error) {
	if runtime.GOOS == "linux" {
		return ph.processesFromProc(systemMemorySize)
	}
	return ph.processesFromPS(systemMemorySize)
}

func getProcLongState(shortState byte) string {
	switch shortState {
	case 'R':
		return "running"
	case 'S':
		return "sleeping"
	case 'D':
		return "blocked"
	case 'Z':
		return "zombie"
	case 'X':
		return "dead"
	case 'T', 't':
		return "stopped"
	case 'W':
		return "paging"
	case 'I':
		return "idle"
	default:
		return fmt.Sprintf("unknown(%s)", string(shortState))
	}
}

// get process states from /proc/(pid)/stat
func (ph *ProcessHandler) processesFromProc(systemMemorySize uint64) ([]*ProcStat, error) {
	filepaths, err := filepath.Glob(helper.HostProc() + "/[0-9]*/status")
	if err != nil {
		return nil, err
	}

	var procs []*ProcStat
	var updatedProcessCache = make(map[int]*process.Process)

	for _, statusFilepath := range filepaths {
		statusFile, err := readProcFile(statusFilepath)
		if err != nil {
			if err != errorProcessTerminated {
				ph.logger.Errorf("readProcFile error:%v", err)
			}
			continue
		}

		parsedProcStatus := ph.parseProcStatusFile(statusFile)
		stat := &ProcStat{ParentPID: parsedProcStatus.PPID, State: parsedProcStatus.State}
		// get the PID from the filepath(/proc/<pid>/status) itself
		pathParts := strings.Split(statusFilepath, string(filepath.Separator))
		pidString := pathParts[len(pathParts)-2]
		stat.PID, err = strconv.Atoi(pidString)
		if err != nil {
			ph.logger.Errorf("proc/status: failed to convert PID(%s) to int:%v", pidString, err)
		}

		commFilepath := helper.HostProc() + "/" + pidString + "/comm"
		comm, err := readProcFile(commFilepath)
		if err != nil && err != errorProcessTerminated {
			ph.logger.Errorf("failed to read comm(%s):%v", commFilepath, err)
		} else if err == nil {
			stat.Name = string(bytes.TrimRight(comm, "\n"))
		}

		cmdLineFilepath := helper.HostProc() + "/" + pidString + "/cmdline"
		cmdline, err := readProcFile(cmdLineFilepath)
		if err != nil && err != errorProcessTerminated {
			ph.logger.Errorf("failed to read cmdline(%s):%v", cmdLineFilepath, err)
		} else if err == nil {
			stat.Cmdline = strings.Replace(string(bytes.TrimRight(cmdline, "\x00")), "\x00", " ", -1)
		}

		cgroupFilepath := helper.HostProc() + "/" + pidString + "/cgroup"
		cgroup, err := readProcFile(cgroupFilepath)
		if err != nil && err != errorProcessTerminated {
			ph.logger.Errorf("failed to read cgroup(%s):%v", cgroupFilepath, err)
		} else if err == nil {
			reParts := dockerContainerIDRE.FindStringSubmatch(string(cgroup))
			if len(reParts) > 0 {
				containerID := reParts[1]
				containerName, err := ph.dockerHandler.ContainerNameByID(containerID)
				if err != nil {
					if err != docker.ErrorNotImplementedForOS && err != docker.ErrorDockerNotAvailable {
						ph.logger.Errorf("failed to read docker container name by id(%s):%v", containerID, err)
					}
				} else {
					stat.Container = containerName
				}
			}
		}

		statFilepath := helper.HostProc() + "/" + pidString + "/stat"
		statFileContent, err := readProcFile(statFilepath)
		if err != nil && err != errorProcessTerminated {
			ph.logger.Errorf("failed to read stat (%s):%v", statFilepath, err)
		} else if err == nil {
			stat.ProcessGID = ph.parseProcessGroupIDFromStatFile(statFileContent)
		}

		if stat.PID > 0 {
			p := ph.getProcessByPID(stat.PID)
			stat.RSS, stat.VMS, stat.MemoryUsagePercent, stat.CPUAverageUsagePercent = ph.gatherProcessResourceUsage(p, systemMemorySize)
			updatedProcessCache[stat.PID] = p
		}

		procs = append(procs, stat)
	}

	ph.processCache.monitoredProcessCache = updatedProcessCache

	return procs, nil
}

func (ph *ProcessHandler) parseProcStatusFile(b []byte) procStatus {
	// fill default value
	// we need non-zero values in order to check if we set them, because PPID can be 0
	status := procStatus{
		PPID: -1,
	}

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		switch strings.ToLower(fields[0]) {
		case "ppid:":
			var err error
			status.PPID, err = strconv.Atoi(fields[1])
			if err != nil {
				ph.logger.Errorf("proc/status: failed to convert PPID(%s) to int:%v", fields[1], err)
			}
		case "state:":
			// extract raw long state
			// eg "State:	S (sleeping)"
			if len(fields) >= 3 {
				status.State = strings.ToLower(strings.Trim(fields[2], "()"))
				break
			}

			if len(fields) < 2 {
				break
			}
			// determine long state from the short one in case long one is not available
			// eg "State:	S"
			status.State = getProcLongState(fields[1][0])
		}

		if status.PPID >= 0 && status.State != "" {
			// we found all fields we want to
			// we can break and return
			break
		}
	}

	return status
}

func (ph *ProcessHandler) parseProcessGroupIDFromStatFile(b []byte) int {
	statFields := procPidStatSplit(string(b))
	resultStr := statFields[4]
	if resultStr == "" {
		ph.logger.Debugf("proc/stat: could not parse stat file: %s", string(b))
		return -1
	}

	pgrp, err := strconv.ParseInt(resultStr, 10, 32)
	if err != nil {
		ph.logger.Errorf("proc/stat: failed to convert PGRP (%s) to int. Stat file: %s:%v", resultStr, string(b), err)
		return -1
	}
	return int(pgrp)
}

// procPidStatSplit tries to parse /proc/<pid>/stat file
// from uber-archive/cpustat
// You might think that we could split on space, but due to what can at best be called
// a shortcoming of the /proc/pid/stat format, the comm field can have unescaped spaces, parens, etc.
// This may be a bit paranoid, because even many common tools like htop do not handle this case well.
func procPidStatSplit(b string) []string {
	line := strings.TrimSpace(b)

	var splitParts = make([]string, 52)

	partnum := 0
	strpos := 0
	start := 0
	inword := false
	space := " "[0]
	openParen := "("[0]
	closeParen := ")"[0]
	groupchar := space

	for ; strpos < len(line); strpos++ {
		if inword {
			if line[strpos] == space && (groupchar == space || line[strpos-1] == groupchar) {
				splitParts[partnum] = line[start:strpos]
				partnum++
				start = strpos
				inword = false
			}
		} else {
			if line[strpos] == openParen {
				groupchar = closeParen
				inword = true
				start = strpos
				strpos = strings.LastIndex(line, ")") - 1
				if strpos <= start { // if we can't parse this insane field, skip to the end
					strpos = len(line)
					inword = false
				}
			} else if line[strpos] != space {
				groupchar = space
				inword = true
				start = strpos
			}
		}
	}

	if inword {
		splitParts[partnum] = line[start:strpos]
		partnum++
	}

	for ; partnum < 52; partnum++ {
		splitParts[partnum] = ""
	}
	return splitParts
}

func readProcFile(filename string) ([]byte, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		// if file doesn't exists it means that process was closed after we got the directory listing
		if os.IsNotExist(err) {
			return nil, errorProcessTerminated
		}

		// Reading from /proc/<PID> fails with ESRCH if the process has
		// been terminated between open() and read().
		if perr, ok := err.(*os.PathError); ok && perr.Err == syscall.ESRCH {
			return nil, errorProcessTerminated
		}

		return nil, err
	}

	return data, nil
}

func execPS() ([]byte, error) {
	bin, err := exec.LookPath("ps")
	if err != nil {
		return nil, err
	}

	out, _ := exec.Command(bin, "axwwo", "pid,ppid,pgrp,state,command").Output()
	return out, nil
}

func (ph *ProcessHandler) processesFromPS(systemMemorySize uint64) ([]*ProcStat, error) {
	out, err := execPS()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var procs []*ProcStat
	var updatedProcessCache = make(map[int]*process.Process)
	var columnsIndex = map[string]int{}

	for i, line := range lines {
		parts := strings.Fields(line)

		if i == 0 {
			// parse the header
			for colIndex, colName := range parts {
				columnsIndex[strings.ToUpper(colName)] = colIndex
			}
			continue
		}

		if len(parts) < 3 {
			continue
		}

		stat := &ProcStat{}

		if pidIndex, exists := columnsIndex["PID"]; exists {
			pidString := parts[pidIndex]
			stat.PID, err = strconv.Atoi(pidString)
			if err != nil {
				ph.logger.Errorf("ps: failed to convert PID(%s) to int:%v", pidString, err)
			}
		} else {
			// we can't set PID set to default 0 if it is unavailable for some reason, because 0 PID means the kernel(Swapper) process
			stat.PID = -1
		}

		if ppidIndex, exists := columnsIndex["PPID"]; exists {
			ppidString := parts[ppidIndex]
			stat.ParentPID, err = strconv.Atoi(ppidString)
			if err != nil {
				ph.logger.Errorf("ps: failed to convert PPID(%s) to int:%v", ppidString, err)
			}
		} else {
			// we can't left ParentPID set to default 0 if it is unavailable for some reason, because 0 PID means the kernel task process
			stat.ParentPID = -1
		}

		if pgidIndex, exists := columnsIndex["PGRP"]; exists {
			pgidString := parts[pgidIndex]
			stat.ProcessGID, err = strconv.Atoi(pgidString)
			if err != nil {
				ph.logger.Errorf("ps: failed to convert PGID(%s) to int:%v", pgidString, err)
			}
		} else {
			// we can't left ProcessGID set to default 0 if it is unavailable for some reason, because 0 PGID means the kernel task group
			stat.ProcessGID = -1
		}

		if statIndex, exists := columnsIndex["STAT"]; exists {
			stat.State = getProcLongState(parts[statIndex][0])
		}

		// COMMAND must be the last column otherwise we can't parse it because it can contains spaces
		if commandIndex, exists := columnsIndex["COMMAND"]; exists && commandIndex == (len(columnsIndex)-1) {
			stat.Cmdline = strings.Join(parts[commandIndex:], " ")

			// extract the executable name without the arguments
			fileBaseWithArgs := filepath.Base(stat.Cmdline)
			fileBaseParts := strings.Fields(fileBaseWithArgs)
			stat.Name = fileBaseParts[0]
		}

		if stat.PID > 0 {
			p := ph.getProcessByPID(stat.PID)
			stat.RSS, stat.VMS, stat.MemoryUsagePercent, stat.CPUAverageUsagePercent = ph.gatherProcessResourceUsage(p, systemMemorySize)
			updatedProcessCache[stat.PID] = p
		}

		procs = append(procs, stat)
	}
	ph.processCache.monitoredProcessCache = updatedProcessCache

	return procs, nil
}

func (ph *ProcessHandler) getProcessByPID(pid int) *process.Process {
	p, exists := ph.processCache.monitoredProcessCache[pid]
	if !exists {
		p = &process.Process{
			Pid: int32(pid),
		}
	}
	return p
}

func (ph *ProcessHandler) gatherProcessResourceUsage(proc *process.Process, systemMemorySize uint64) (uint64, uint64, float32, float32) {
	memoryInfo, err := proc.MemoryInfo()
	if err != nil {
		ph.logger.Errorf("failed to get memory info:%v", err)
		return 0, 0, 0.0, 0.0
	}
	memUsagePercent := 0.0
	if systemMemorySize > 0 {
		memUsagePercent = (float64(memoryInfo.RSS) / float64(systemMemorySize)) * 100
	}

	// side effect: p.Percent() call update process internally
	cpuUsagePercent, err := proc.Percent(time.Duration(0))
	if err != nil {
		ph.logger.Errorf("failed to get CPU usage:%v", err)
	}

	return memoryInfo.RSS, memoryInfo.VMS, float32(helper.RoundToTwoDecimalPlaces(memUsagePercent)), float32(helper.RoundToTwoDecimalPlaces(cpuUsagePercent))
}

func isKernelTask(procStat *ProcStat) bool {
	return procStat.ParentPID == 0 || procStat.ProcessGID == 0
}
