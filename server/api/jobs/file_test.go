package jobs

import (
	"errors"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/test/jb"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"
)

func TestGetByJID(t *testing.T) {
	job1Running := jb.New(t).Status(models.JobStatusRunning).Build()
	job2Running := jb.New(t).SID(job1Running.SID).Status(models.JobStatusRunning).Build()
	ft := time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC)
	job3 := jb.New(t).Status(models.JobStatusSuccessful).FinishedAt(ft).Build()

	testCases := []struct {
		name string

		runningJobs          []*models.Job
		sid                  string
		jid                  string
		fReturnExist         bool
		fReturnExistErr      error
		fReturnReadFileErr   error
		fSetReadFileDestFunc func(dest interface{})

		wantExistPath   bool
		wantReadFile    bool
		wantJob         *models.Job
		wantErrContains string
	}{
		{
			name:        "running job",
			runningJobs: []*models.Job{job1Running},
			sid:         job1Running.SID,
			jid:         job1Running.JID,
			wantJob:     job1Running,
		},
		{
			name:          "running job with different JID",
			runningJobs:   []*models.Job{job2Running},
			sid:           job2Running.SID,
			jid:           job1Running.JID,
			wantJob:       nil,
			wantExistPath: true,
		},
		{
			name:          "job file not exist",
			sid:           job3.SID,
			jid:           job3.JID,
			wantJob:       nil,
			wantExistPath: true,
		},
		{
			name:            "error on file exist check",
			sid:             job3.SID,
			jid:             job3.JID,
			fReturnExistErr: errors.New("fake exist check error"),
			wantJob:         nil,
			wantExistPath:   true,
			wantErrContains: "fake exist check error",
		},
		{
			name:         "job file exist",
			sid:          job3.SID,
			jid:          job3.JID,
			fReturnExist: true,
			fSetReadFileDestFunc: func(dest interface{}) {
				d := dest.(*models.Job)
				*d = *job3
			},
			wantJob:       job3,
			wantExistPath: true,
			wantReadFile:  true,
		},
		{
			name:               "error on read file",
			sid:                job3.SID,
			jid:                job3.JID,
			fReturnExist:       true,
			fReturnReadFileErr: errors.New("fake read file error"),
			wantJob:            nil,
			wantExistPath:      true,
			wantReadFile:       true,
			wantErrContains:    "fake read file error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			// given
			fileAPIMock := test.NewFileAPIMock()
			fileAPIMock.ReturnExist = tc.fReturnExist
			fileAPIMock.ReturnExistErr = tc.fReturnExistErr
			fileAPIMock.ReturnReadFileErr = tc.fReturnReadFileErr
			fileAPIMock.SetReadFileDestFunc = tc.fSetReadFileDestFunc

			fp := NewFileProvider(fileAPIMock, "jobs-dir")
			if len(tc.runningJobs) > 0 {
				for _, cur := range tc.runningJobs {
					fp.runningJobs.Set(cur.SID, cur)
				}
			}

			// when
			gotJob, gotErr := fp.GetByJID(tc.sid, tc.jid)

			// then
			// check result
			if len(tc.wantErrContains) > 0 {
				require.Error(t, gotErr)
				assert.Contains(gotErr.Error(), tc.wantErrContains)
			} else {
				assert.NoError(gotErr)
				assert.Equal(tc.wantJob, gotJob)
			}
			// check invoked funcs and input values
			assert.Equal(tc.wantExistPath, fileAPIMock.ExistPathInvoked)
			var wantFilePathPrefix string
			var wantFilePathSuffix string
			if tc.wantExistPath || tc.wantReadFile {
				wantFilePathPrefix = fp.getSessionDirFullPath(tc.sid)
				wantFilePathSuffix = tc.jid + jobFileExtension
			}
			if tc.wantExistPath {
				assert.Contains(fileAPIMock.InputExistPath, wantFilePathPrefix)
				assert.Contains(fileAPIMock.InputExistPath, wantFilePathSuffix)
			}
			assert.Equal(tc.wantReadFile, fileAPIMock.ReadFileInvoked)
			if tc.wantReadFile {
				assert.Contains(fileAPIMock.InputReadFile, wantFilePathPrefix)
				assert.Contains(fileAPIMock.InputReadFile, wantFilePathSuffix)
			}
		})
	}
}

func TestGetSummariesBySID(t *testing.T) {
	job1Running := jb.New(t).Status(models.JobStatusRunning).Build()
	job2Running := jb.New(t).Status(models.JobStatusRunning).Build() // with different sid
	ft := time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC)
	job1 := jb.New(t).SID(job1Running.SID).Status(models.JobStatusSuccessful).FinishedAt(ft).Build()
	job2 := jb.New(t).SID(job1Running.SID).Status(models.JobStatusUnknown).FinishedAt(ft.Add(time.Minute)).Build()
	job3 := jb.New(t).SID(job1Running.SID).Status(models.JobStatusFailed).FinishedAt(ft.Add(-time.Hour)).Build()
	job4 := jb.New(t).SID(job1Running.SID).Status(models.JobStatusSuccessful).FinishedAt(ft).Build()
	job5 := jb.New(t).SID(job1Running.SID).Status(models.JobStatusSuccessful).FinishedAt(ft).Build()
	file1 := &test.FileMock{
		ReturnName:    "s-" + job1.JID + ".json",
		ReturnModTime: *job1.FinishedAt,
	}
	file2 := &test.FileMock{
		ReturnName:    "u-" + job2.JID + ".json",
		ReturnModTime: *job2.FinishedAt,
	}
	file3 := &test.FileMock{
		ReturnName:    "f-" + job3.JID + ".json",
		ReturnModTime: *job3.FinishedAt,
	}
	file4 := &test.FileMock{
		ReturnName:    "s-" + job4.JID + ".data", // wrong extension
		ReturnModTime: *job4.FinishedAt,
	}
	file5 := &test.FileMock{
		ReturnName:    "s-" + job5.JID, // no extension
		ReturnModTime: *job5.FinishedAt,
	}
	file6 := &test.FileMock{
		ReturnName:    "p-" + job5.JID, // incorrect prefix
		ReturnModTime: *job5.FinishedAt,
	}
	file7 := &test.FileMock{
		ReturnName:    job5.JID, // no prefix
		ReturnModTime: *job5.FinishedAt,
	}

	testCases := []struct {
		name string

		runningJobs          []*models.Job
		sid                  string
		fReturnExist         bool
		fReturnExistErr      error
		fReturnReadDirErr    error
		fReturnReadDirFiles  []os.FileInfo
		fSetReadFileDestFunc func(dest interface{})

		wantExistPath   bool
		wantReadDir     bool
		wantRes         []*models.JobSummary
		wantErrContains string
	}{
		{
			name:          "no jobs",
			runningJobs:   nil,
			sid:           job1Running.SID,
			fReturnExist:  true,
			wantRes:       []*models.JobSummary{},
			wantExistPath: true,
			wantReadDir:   true,
		},
		{
			name:          "jobs dir not exist",
			sid:           job1Running.SID,
			fReturnExist:  false,
			wantRes:       []*models.JobSummary{},
			wantExistPath: true,
			wantReadDir:   false,
		},
		{
			name:                "only running job",
			runningJobs:         []*models.Job{job1Running, job2Running},
			sid:                 job1Running.SID,
			fReturnReadDirFiles: nil,
			wantRes:             []*models.JobSummary{&job1Running.JobSummary},
			wantExistPath:       true,
		},
		{
			name:                "only finished jobs",
			runningJobs:         nil,
			sid:                 job1Running.SID,
			fReturnExist:        true,
			fReturnReadDirFiles: []os.FileInfo{file1, file2, file3, file4, file5, file6, file7},
			wantRes:             []*models.JobSummary{&job1.JobSummary, &job2.JobSummary, &job3.JobSummary},
			wantExistPath:       true,
			wantReadDir:         true,
		},
		{
			name:                "running and finished jobs",
			runningJobs:         []*models.Job{job1Running, job2Running},
			sid:                 job1Running.SID,
			fReturnExist:        true,
			fReturnReadDirFiles: []os.FileInfo{file1, file2, file3},
			wantRes:             []*models.JobSummary{&job1Running.JobSummary, &job1.JobSummary, &job2.JobSummary, &job3.JobSummary},
			wantExistPath:       true,
			wantReadDir:         true,
		},
		{
			name:            "error on dir exist check",
			sid:             job1Running.SID,
			fReturnExistErr: errors.New("fake exist check error"),
			wantRes:         []*models.JobSummary{},
			wantExistPath:   true,
			wantErrContains: "fake exist check error",
		},
		{
			name:              "error on read dir",
			sid:               job1Running.SID,
			fReturnExist:      true,
			fReturnReadDirErr: errors.New("fake read dir error"),
			wantRes:           []*models.JobSummary{},
			wantExistPath:     true,
			wantReadDir:       true,
			wantErrContains:   "fake read dir error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			// given
			fileAPIMock := test.NewFileAPIMock()
			fileAPIMock.ReturnExist = tc.fReturnExist
			fileAPIMock.ReturnExistErr = tc.fReturnExistErr
			fileAPIMock.ReturnReadDirErr = tc.fReturnReadDirErr
			fileAPIMock.ReturnReadDirFiles = tc.fReturnReadDirFiles

			fp := NewFileProvider(fileAPIMock, "jobs-dir")
			if len(tc.runningJobs) > 0 {
				for _, cur := range tc.runningJobs {
					fp.runningJobs.Set(cur.SID, cur)
				}
			}

			// when
			gotRes, gotErr := fp.GetSummariesBySID(tc.sid)

			// then
			// check result
			if len(tc.wantErrContains) > 0 {
				require.Error(t, gotErr)
				assert.Contains(gotErr.Error(), tc.wantErrContains)
			} else {
				assert.NoError(gotErr)
				assert.ElementsMatch(tc.wantRes, gotRes)
			}
			// check invoked funcs and input values
			assert.Equal(tc.wantExistPath, fileAPIMock.ExistPathInvoked)
			var wantDirPath string
			if tc.wantExistPath || tc.wantReadDir {
				wantDirPath = fp.getSessionDirFullPath(tc.sid)
			}
			if tc.wantExistPath {
				assert.Equal(wantDirPath, fileAPIMock.InputExistPath)
			}
			assert.Equal(tc.wantReadDir, fileAPIMock.ReadDirInvoked)
			if tc.wantReadDir {
				assert.Equal(wantDirPath, fileAPIMock.InputReadDir)
			}
		})
	}
}

func TestSaveJob(t *testing.T) {
	job1Running := jb.New(t).Status(models.JobStatusRunning).Build()
	job2Running := jb.New(t).SID(job1Running.SID).Status(models.JobStatusRunning).Build()
	job3Running := jb.New(t).Status(models.JobStatusRunning).Build() // with different sid
	ft := time.Date(2020, 10, 10, 10, 10, 10, 0, time.UTC)
	job1 := jb.New(t).SID(job1Running.SID).JID(job1Running.JID).Status(models.JobStatusSuccessful).FinishedAt(ft).Build()
	job2 := jb.New(t).SID(job1Running.SID).Status(models.JobStatusUnknown).FinishedAt(ft).Build() // with same SID but different JID

	testCases := []struct {
		name string

		runningJobs          []*models.Job
		job                  *models.Job
		fReturnMakeDirErr    error
		fReturnCreateFileErr error

		wantRunningJobs []*models.Job
		wantMakeDir     bool
		wantCreateFile  bool
		wantErrContains string
	}{
		{
			name: "nil job",
			job:  nil,
		},
		{
			name:            "running job",
			job:             job1Running,
			wantRunningJobs: []*models.Job{job1Running},
		},
		{
			name:            "running job replaces previous running job",
			runningJobs:     []*models.Job{job1Running},
			job:             job2Running,
			wantRunningJobs: []*models.Job{job2Running},
		},
		{
			name:            "running jobs",
			runningJobs:     []*models.Job{job1Running},
			job:             job3Running,
			wantRunningJobs: []*models.Job{job1Running, job3Running},
		},
		{
			name:            "finished job",
			job:             job1,
			wantRunningJobs: []*models.Job{},
			wantMakeDir:     true,
			wantCreateFile:  true,
		},
		{
			name:            "finished job replaces running job",
			runningJobs:     []*models.Job{job1Running, job3Running},
			job:             job1,
			wantRunningJobs: []*models.Job{job3Running},
			wantMakeDir:     true,
			wantCreateFile:  true,
		},
		{
			name:            "finished job does not replace running job wit the same SID but different JID",
			runningJobs:     []*models.Job{job1Running, job3Running},
			job:             job2,
			wantRunningJobs: []*models.Job{job1Running, job3Running},
			wantMakeDir:     true,
			wantCreateFile:  true,
		},
		{
			name:              "error on make dir",
			runningJobs:       []*models.Job{job1Running, job3Running},
			job:               job1,
			fReturnMakeDirErr: errors.New("fake make dir error"),
			wantMakeDir:       true,
			wantErrContains:   "fake make dir error",
			wantRunningJobs:   []*models.Job{job3Running},
		},
		{
			name:                 "error on create file",
			runningJobs:          []*models.Job{job1Running, job3Running},
			job:                  job1,
			fReturnCreateFileErr: errors.New("fake create file error"),
			wantMakeDir:          true,
			wantCreateFile:       true,
			wantErrContains:      "fake create file error",
			wantRunningJobs:      []*models.Job{job3Running},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			// given
			fileAPIMock := test.NewFileAPIMock()
			fileAPIMock.ReturnMakeDirErr = tc.fReturnMakeDirErr
			fileAPIMock.ReturnCreateFileErr = tc.fReturnCreateFileErr

			fp := NewFileProvider(fileAPIMock, "jobs-dir")
			if len(tc.runningJobs) > 0 {
				for _, cur := range tc.runningJobs {
					fp.runningJobs.Set(cur.SID, cur)
				}
			}

			// when
			gotErr := fp.SaveJob(tc.job)

			// then
			// check result
			if len(tc.wantErrContains) > 0 {
				require.Error(t, gotErr)
				assert.Contains(gotErr.Error(), tc.wantErrContains)
			} else {
				assert.NoError(gotErr)
			}
			assert.ElementsMatch(tc.wantRunningJobs, fp.runningJobs.GetAll())

			// check invoked funcs and input values
			assert.Equal(tc.wantMakeDir, fileAPIMock.MakeDirInvoked)
			if tc.wantMakeDir {
				wantDirPath := fp.getSessionDirFullPath(tc.job.SID)
				assert.Equal(wantDirPath, fileAPIMock.InputMakeDir)
			}
			if tc.wantCreateFile {
				wantFilePath := fp.getFileFullPath(tc.job.SID, tc.job.JID, tc.job.Status)
				assert.Equal(wantFilePath, fileAPIMock.InputCreateFile)
				assert.Equal(tc.job, fileAPIMock.InputCreateFileContent)
			}
		})
	}
}

func TestGetFileFullPath(t *testing.T) {
	// given
	testJobsDir := "jobs-dir"
	testJID := "456"
	sidSHA1 := "40bd001563085fc35165329ea1ff5c5ecbdbbeef"
	fp := NewFileProvider(test.NewFileAPIMock(), testJobsDir)

	testCases := []struct {
		name       string
		status     string
		wantPrefix string
	}{
		{
			name:       "empty status",
			status:     "",
			wantPrefix: "",
		},
		{
			name:       "unsupported status",
			status:     "unsupported",
			wantPrefix: "",
		},
		{
			name:       "successful status",
			status:     models.JobStatusSuccessful,
			wantPrefix: "s-",
		},
		{
			name:       "failed status",
			status:     models.JobStatusFailed,
			wantPrefix: "f-",
		},
		{
			name:       "unknown status",
			status:     models.JobStatusUnknown,
			wantPrefix: "u-",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotRes := fp.getFileFullPath("123", testJID, tc.status)

			// then
			assert.Equal(t, path.Join(testJobsDir, sidSHA1, tc.wantPrefix+testJID+".json"), gotRes)
		})
	}
}
