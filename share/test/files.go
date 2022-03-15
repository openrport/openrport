package test

import (
	"crypto/md5"
	"io"
	"os"
	"time"

	"github.com/stretchr/testify/mock"
)

type FileAPIMock struct {
	mock.Mock
	SetReadFileDestFunc func(dest interface{})
}

func NewFileAPIMock() *FileAPIMock {
	return &FileAPIMock{}
}

func (f *FileAPIMock) ReadDir(dir string) ([]os.FileInfo, error) {
	args := f.Called(dir)

	return args.Get(0).([]os.FileInfo), args.Error(1)
}

func (f *FileAPIMock) MakeDirAll(dir string) error {
	args := f.Called(dir)

	return args.Error(0)
}

func (f *FileAPIMock) WriteJSON(file string, content interface{}) error {
	args := f.Called(file, content)

	return args.Error(0)
}

func (f *FileAPIMock) ReadJSON(file string, dest interface{}) error {
	args := f.Called(file, dest)

	if f.SetReadFileDestFunc != nil {
		f.SetReadFileDestFunc(dest)
	}

	return args.Error(0)
}

func (f *FileAPIMock) Exist(path string) (bool, error) {
	args := f.Called(path)
	return args.Bool(0), args.Error(1)
}

func (f *FileAPIMock) Open(file string) (io.ReadWriteCloser, error) {
	args := f.Called(file)
	return args.Get(0).(io.ReadWriteCloser), args.Error(1)
}

func (f *FileAPIMock) Write(fileName string, content string) error {
	args := f.Called(fileName, content)

	return args.Error(0)
}

func (f *FileAPIMock) CreateFile(path string, sourceReader io.Reader) (writtenBytes int64, err error) {
	args := f.Called(path, sourceReader)

	return args.Get(0).(int64), args.Error(1)
}

func (f *FileAPIMock) ChangeOwner(path, owner, group string) error {
	args := f.Called(path, owner, group)

	return args.Error(0)
}

func (f *FileAPIMock) ChangeMode(path string, targetMode os.FileMode) error {
	args := f.Called(path, targetMode)

	return args.Error(0)
}

func (f *FileAPIMock) Remove(name string) error {
	args := f.Called(name)

	return args.Error(0)
}

func (f *FileAPIMock) Rename(oldPath, newPath string) error {
	args := f.Called(oldPath, newPath)

	return args.Error(0)
}

func (f *FileAPIMock) CreateDirIfNotExists(path string, mode os.FileMode) (wasCreated bool, err error) {
	args := f.Called(path, mode)

	return args.Bool(0), args.Error(1)
}

func (f *FileAPIMock) GetFileMode(file string) (os.FileMode, error) {
	args := f.Called(file)

	return args.Get(0).(os.FileMode), args.Error(1)
}

func (f *FileAPIMock) GetFileOwnerAndGroup(file string) (uid, gid uint32, err error) {
	args := f.Called(file)

	return args.Get(0).(uint32), args.Get(1).(uint32), args.Error(2)
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

func Md5Hash(input string) []byte {
	hashSum := md5.Sum([]byte(input))

	return hashSum[:]
}
