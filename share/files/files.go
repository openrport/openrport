package files

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	errors2 "github.com/pkg/errors"
)

const DefaultUploadTempFolder = "filepush"

const DefaultMode = os.FileMode(0764)

type FileAPI interface {
	ReadDir(dir string) ([]os.FileInfo, error)
	MakeDirAll(dir string) error
	WriteJSON(file string, content interface{}) error
	Write(file string, content string) error
	ReadJSON(file string, dest interface{}) error
	Open(file string) (io.ReadWriteCloser, error)
	Exist(path string) (bool, error)
	CreateFile(path string, sourceReader io.Reader) (writtenBytes int64, err error)
	ChangeOwner(path, owner, group string) error
	ChangeMode(path string, targetMode os.FileMode) error
	CreateDirIfNotExists(path string, mode os.FileMode) (wasCreated bool, err error)
	Remove(name string) error
	Rename(oldPath, newPath string) error
	GetFileMode(file string) (os.FileMode, error)
	GetFileOwnerAndGroup(file string) (uid, gid uint32, err error)
}

type FileSystem struct {
}

func NewFileSystem() *FileSystem {
	return &FileSystem{}
}

// ReadDir reads the given directory and returns a list of directory entries sorted by filename.
func (f *FileSystem) ReadDir(dir string) ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %s", err)
	}
	return files, nil
}

// MakeDirAll creates a given directory along with any necessary parents.
// If path is already a directory, it does nothing and returns nil.
// It is created with mode 0777.
func (f *FileSystem) MakeDirAll(dir string) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create dir %q: %s", dir, err)
	}
	return nil
}

// WriteJSON creates or truncates a given file and writes a given content to it as JSON
// with indentation. If the file does not exist, it is created with mode 0666.
func (f *FileSystem) WriteJSON(fileName string, content interface{}) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %s", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "	")
	if err := encoder.Encode(content); err != nil {
		return fmt.Errorf("failed to write data to file: %v", err)
	}

	return nil
}

// Write creates or truncates a given file and writes a given content to it.
// If the file does not exist, it is created with mode 0666.
func (f *FileSystem) Write(fileName string, content string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %s", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write data to file: %v", err)
	}

	return nil
}

// ReadJSON reads a given file and stores the parsed content into a destination value.
// A successful call returns err == nil, not err == EOF.
func (f *FileSystem) ReadJSON(file string, dest interface{}) error {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read data from file: %s", err)
	}

	err = json.Unmarshal(b, dest)
	if err != nil {
		return fmt.Errorf("failed to decode data into %T: %s", dest, err)
	}

	return nil
}

func (f *FileSystem) Open(file string) (io.ReadWriteCloser, error) {
	return os.Open(file)
}

func (f *FileSystem) GetFileMode(file string) (os.FileMode, error) {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return 0, err
	}

	return fileInfo.Mode(), nil
}

func (f *FileSystem) GetFileOwnerAndGroup(file string) (uid, gid uint32, err error) {
	return GetFileUIDAndGID(file)
}

// Exist returns a boolean indicating whether a file or directory with a given path exists.
func (f *FileSystem) Exist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (f *FileSystem) CreateDirIfNotExists(path string, mode os.FileMode) (wasCreated bool, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, errors.New("data directory cannot be empty")
	}

	dirExists, err := f.Exist(path)
	if err != nil {
		return false, errors2.Wrapf(err, "failed to read folder info %s", path)
	}

	if !dirExists {
		err := os.MkdirAll(path, mode)
		if err != nil {
			return false, errors2.Wrapf(err, "failed to create folder %s", path)
		}
		wasCreated = true
	}

	return wasCreated, nil
}

func (f *FileSystem) ChangeOwner(path, owner, group string) error {
	return ChangeOwner(path, owner, group)
}

func (f *FileSystem) ChangeMode(path string, targetMode os.FileMode) error {
	if targetMode == 0 {
		return nil
	}

	fileStat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if targetMode != fileStat.Mode() {
		return os.Chmod(path, targetMode)
	}

	return nil
}

func (f *FileSystem) CreateFile(path string, sourceReader io.Reader) (writtenBytes int64, err error) {
	targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, DefaultMode)
	if err != nil {
		return 0, err
	}
	defer targetFile.Close()

	copiedBytes, err := io.Copy(targetFile, sourceReader)
	if err != nil {
		return 0, err
	}

	return copiedBytes, nil
}

func (f *FileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (f *FileSystem) Rename(oldPath, newPath string) error {
	return Rename(oldPath, newPath)
}

func Md5HashFromReader(source io.Reader) (hashSum []byte, err error) {
	md5Hash := md5.New()
	_, err = io.Copy(md5Hash, source)
	if err != nil {
		return nil, errors2.Wrapf(err, "failed to calculate md5 checksum")
	}

	return md5Hash.Sum(nil), nil
}

func Md5HashMatch(expectedHashSum []byte, path string, fileAPI FileAPI) (match bool, err error) {
	file, err := fileAPI.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	destinationMd5Hash, err := Md5HashFromReader(file)
	if err != nil {
		return false, err
	}

	if bytes.Equal(expectedHashSum, destinationMd5Hash) {
		return true, nil
	}

	return false, nil
}
