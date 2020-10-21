package jobs

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/models"
)

const jobFileExtension = ".json"

const prefixLength = 2

var prefixStatusMapping = map[string]string{
	"s-": models.JobStatusSuccessful,
	"f-": models.JobStatusFailed,
	"u-": models.JobStatusUnknown,
}

var statusPrefixMapping = map[string]string{
	models.JobStatusSuccessful: "s-",
	models.JobStatusFailed:     "f-",
	models.JobStatusUnknown:    "u-",
}

type FileProvider struct {
	fileAPI     files.FileAPI
	jobsDir     string
	runningJobs *JobCache
}

func NewFileProvider(fileAPI files.FileAPI, jobsDir string) *FileProvider {
	if jobsDir == "" || fileAPI == nil {
		return nil
	}

	return &FileProvider{
		fileAPI:     fileAPI,
		jobsDir:     jobsDir,
		runningJobs: NewEmptyJobCache(),
	}
}

// GetByJID returns a job by a given session id and a job id.
func (p *FileProvider) GetByJID(sid, jid string) (*models.Job, error) {
	// check among running jobs at first
	runningJob := p.runningJobs.Get(sid)
	if runningJob != nil && runningJob.JID == jid {
		return runningJob, nil
	}

	// check among finished jobs
	var existingFilePath string
	// go through each possible prefix to find a requested job
	for curStatus := range statusPrefixMapping {
		filePath := p.getFileFullPath(sid, jid, curStatus)
		exist, err := p.fileAPI.Exist(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to check whether job file exists: %s", err)
		}
		if exist {
			existingFilePath = filePath
			break
		}
	}
	if existingFilePath == "" {
		return nil, nil
	}

	job := models.Job{}
	err := p.fileAPI.ReadFileJSON(existingFilePath, &job)
	if err != nil {
		return nil, fmt.Errorf("failed to parse job file: %s", err)
	}

	return &job, nil
}

// GetSummariesBySID returns short info about all jobs for a given session id.
func (p *FileProvider) GetSummariesBySID(sid string) ([]*models.JobSummary, error) {
	// get finished jobs from the dir
	finishedJobs, err := p.getSummariesFromDir(sid)
	if err != nil {
		return nil, err
	}

	// get a running job from the cache
	runningJob := p.runningJobs.Get(sid)

	var res []*models.JobSummary
	res = append(res, finishedJobs...)
	if runningJob != nil {
		res = append(res, &runningJob.JobSummary)
	}
	return res, nil
}

// getSummariesFromDir returns short info about all finished jobs for a given session id that are stored in files on disk.
func (p *FileProvider) getSummariesFromDir(sid string) ([]*models.JobSummary, error) {
	jobsDir := p.getSessionDirFullPath(sid)
	if exist, err := p.fileAPI.Exist(jobsDir); err != nil {
		return nil, fmt.Errorf("failed to check whether jobs folder %q exists: %s", jobsDir, err)
	} else if !exist {
		return nil, nil
	}

	jobFiles, err := p.fileAPI.ReadDir(jobsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list job files: %s", err)
	}

	res := make([]*models.JobSummary, 0, len(jobFiles))
	for _, jobFile := range jobFiles {
		cur := convertToJobSummary(jobFile)
		if cur != nil {
			res = append(res, cur)
		}
	}

	return res, nil
}

// SaveJob saves a given job. Running jobs are persisted in memory, finished - in file on disk.
func (p *FileProvider) SaveJob(job *models.Job) error {
	if job == nil {
		return nil
	}

	// running job just store in the cache
	if job.Status == models.JobStatusRunning {
		p.runningJobs.Set(job.SID, job)
		return nil
	}

	// it there is the same job among running - remove it from the cache
	runningJob := p.runningJobs.Get(job.SID)
	if runningJob != nil && runningJob.JID == job.JID {
		p.runningJobs.Delete(job.SID)
	}

	if err := p.fileAPI.MakeDirAll(p.getSessionDirFullPath(job.SID)); err != nil {
		return fmt.Errorf("failed to create jobs dir for client: %s", err)
	}

	if err := p.fileAPI.CreateFileJSON(p.getFileFullPath(job.SID, job.JID, job.Status), job); err != nil {
		return fmt.Errorf("failed to create job file: %s", err)
	}

	return nil
}

func convertToJobSummary(file os.FileInfo) *models.JobSummary {
	if !strings.HasSuffix(file.Name(), jobFileExtension) {
		return nil
	}

	fileName := file.Name()
	prefix := fileName[:prefixLength]
	status := prefixStatusMapping[prefix]
	if status == "" {
		return nil
	}
	jid := fileName[prefixLength : len(fileName)-len(jobFileExtension)]
	finishedAt := file.ModTime()
	return &models.JobSummary{
		JID:        jid,
		Status:     status,
		FinishedAt: &finishedAt,
	}
}

func (p *FileProvider) getFileFullPath(sid, jid, status string) string {
	return path.Join(p.getSessionDirFullPath(sid), statusPrefixMapping[status]+jid+jobFileExtension)
}

func (p *FileProvider) getSessionDirFullPath(sid string) string {
	return path.Join(p.jobsDir, getSessionDirName(sid))
}

// getSessionDirName returns a file name for a given session id.
// A hash of a given sid is used because session id is untrusted user input.
func getSessionDirName(sid string) string {
	hashBytes := sha1.Sum([]byte(sid))
	return hex.EncodeToString(hashBytes[:])
}
