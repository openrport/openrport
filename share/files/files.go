package files

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"strings"

	errors2 "github.com/pkg/errors"

	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

const DefaultUploadTempFolder = "filepush"

const DefaultMode = 0764

type FileAPI interface {
	ReadDir(dir string) ([]os.FileInfo, error)
	MakeDirAll(dir string) error
	WriteJSON(file string, content interface{}) error
	Write(file string, content string) error
	ReadJSON(file string, dest interface{}) error
	Exist(path string) (bool, error)
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

func CreateDirIfNotExists(uploadDir string, mode os.FileMode) (wasCreated bool, err error) {
	uploadDir = strings.TrimSpace(uploadDir)
	if uploadDir == "" {
		return false, errors.New("data directory cannot be empty")
	}

	dirExists, err := FileOrDirExists(uploadDir)
	if err != nil {
		return false, errors2.Wrapf(err, "failed to read folder info %s", uploadDir)
	}

	if !dirExists {
		err := os.MkdirAll(uploadDir, mode)
		if err != nil {
			return false, errors2.Wrapf(err, "failed to create folder %s", uploadDir)
		}
		wasCreated = true
	}

	return wasCreated, nil
}

func FileOrDirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func ChangeOwner(reqPayload *models.UploadedFile, log *logger.Logger) error {
	if reqPayload.DestinationFileOwner == "" && reqPayload.DestinationFileGroup == "" {
		return nil
	}

	targetUserUID := os.Getuid()
	if reqPayload.DestinationFileOwner != "" {
		usr, err := user.Lookup(reqPayload.DestinationFileOwner)
		if err != nil {
			return err
		}
		targetUserUID, err = strconv.Atoi(usr.Uid)
		if err != nil {
			return err
		}
	}

	targetGroupGUID := os.Getgid()
	if reqPayload.DestinationFileGroup != "" {
		gr, err := user.LookupGroup(reqPayload.DestinationFileGroup)
		if err != nil {
			return err
		}
		targetGroupGUID, err = strconv.Atoi(gr.Gid)
		if err != nil {
			return err
		}
	}

	log.Infof(
		"will chown file %s, %s:%s[%d:%d]",
		reqPayload.DestinationPath,
		reqPayload.DestinationFileOwner,
		reqPayload.DestinationFileGroup,
		targetUserUID,
		targetGroupGUID,
	)
	err := os.Chown(reqPayload.DestinationPath, targetUserUID, targetGroupGUID)
	if err != nil {
		return err
	}

	return nil
}

func CopyFileToDestination(targetFilePath string, sourceReader io.Reader, log *logger.Logger) (int64, error) {
	targetFile, err := os.OpenFile(targetFilePath, os.O_WRONLY|os.O_CREATE, DefaultMode)
	if err != nil {
		return 0, err
	}
	defer targetFile.Close()

	// todo delete file after all operations have completed
	copiedBytes, err := io.Copy(targetFile, sourceReader)
	if err != nil {
		return 0, err
	}
	log.Infof("copied %d bytes to: %s", copiedBytes, targetFilePath)

	return copiedBytes, nil
}
