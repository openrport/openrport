package test

import (
	"os"
	"time"
)

type FileAPIMock struct {
	ReadDirInvoked     bool
	InputReadDir       string
	ReturnReadDirFiles []os.FileInfo
	ReturnReadDirErr   error

	MakeDirInvoked   bool
	InputMakeDir     string
	ReturnMakeDirErr error

	CreateFileInvoked      bool
	InputCreateFile        string
	InputCreateFileContent interface{}
	ReturnCreateFileErr    error

	ReadFileInvoked     bool
	InputReadFile       string
	ReturnReadFileErr   error
	SetReadFileDestFunc func(dest interface{})

	ExistPathInvoked bool
	InputExistPath   string
	ReturnExist      bool
	ReturnExistErr   error
}

func NewFileAPIMock() *FileAPIMock {
	return &FileAPIMock{}
}

func (f *FileAPIMock) ReadDir(dir string) ([]os.FileInfo, error) {
	f.ReadDirInvoked = true
	f.InputReadDir = dir
	return f.ReturnReadDirFiles, f.ReturnReadDirErr
}

func (f *FileAPIMock) MakeDirAll(dir string) error {
	f.MakeDirInvoked = true
	f.InputMakeDir = dir
	return f.ReturnMakeDirErr
}

func (f *FileAPIMock) CreateFileJSON(file string, content interface{}) error {
	f.CreateFileInvoked = true
	f.InputCreateFile = file
	f.InputCreateFileContent = content
	return f.ReturnCreateFileErr
}

func (f *FileAPIMock) ReadFileJSON(file string, dest interface{}) error {
	f.ReadFileInvoked = true
	f.InputReadFile = file
	if f.SetReadFileDestFunc != nil {
		f.SetReadFileDestFunc(dest)
	}
	return f.ReturnReadFileErr
}

func (f *FileAPIMock) Exist(path string) (bool, error) {
	f.ExistPathInvoked = true
	f.InputExistPath = path
	return f.ReturnExist, f.ReturnExistErr
}

type FileMock struct {
	os.FileInfo
	ReturnName    string
	ReturnModTime time.Time
}

func (f *FileMock) Name() string {
	return f.ReturnName
}

func (f *FileMock) ModTime() time.Time {
	return f.ReturnModTime
}
