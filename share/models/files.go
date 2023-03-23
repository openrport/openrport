package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/realvnc-labs/rport/share/logger"

	errors2 "github.com/pkg/errors"
)

const (
	uploadedFileDestinationPathKey = "dest"
	uploadedFileOwnerKey           = "user"
	uploadedFileOwnerGroupKey      = "group"
	uploadedFileModeKey            = "mode"
	fileWriteForcedKey             = "force"
	fileSyncdKey                   = "sync"
	IDKey                          = "id"
)

type UploadedFile struct {
	ID                   string
	SourceFilePath       string
	DestinationPath      string
	DestinationFileMode  os.FileMode
	DestinationFileOwner string
	DestinationFileGroup string
	ForceWrite           bool
	Sync                 bool
	Md5Checksum          []byte
}

func (uf UploadedFile) Validate() error {
	if uf.SourceFilePath == "" {
		return errors.New("empty source file name")
	}

	if uf.DestinationPath == "" {
		return errors.New("empty destination file path")
	}

	return nil
}

func (uf UploadedFile) ValidateDestinationPath(globPatters []string, log *logger.Logger) error {
	destinationDir := filepath.Dir(uf.DestinationPath)
	for _, p := range globPatters {
		matchedDir, err := filepath.Match(p, destinationDir)
		if err != nil {
			log.Errorf("failed to match glob pattern %s against destination directory %s: %v", p, uf.DestinationPath, err)
			continue
		}
		if matchedDir {
			return fmt.Errorf("target path %s matches protected pattern %s, therefore the file push request is rejected", destinationDir, p)
		}

		matchedFile, err := filepath.Match(p, uf.DestinationPath)
		if err != nil {
			log.Errorf("failed to match glob pattern %s against file name %s: %v", p, uf.DestinationPath, err)
			continue
		}

		if matchedFile {
			return fmt.Errorf("target path %s matches protected pattern %s, therefore the file push request is rejected", uf.DestinationPath, p)
		}
	}

	return nil
}

func (uf *UploadedFile) FromMultipartRequest(req *http.Request) error {
	var err error
	if req.MultipartForm == nil {
		return nil
	}

	if len(req.MultipartForm.Value[uploadedFileDestinationPathKey]) > 0 {
		uf.DestinationPath = req.MultipartForm.Value[uploadedFileDestinationPathKey][0]
	}

	if len(req.MultipartForm.Value[uploadedFileOwnerKey]) > 0 {
		uf.DestinationFileOwner = req.MultipartForm.Value[uploadedFileOwnerKey][0]
	}

	if len(req.MultipartForm.Value[uploadedFileOwnerGroupKey]) > 0 {
		uf.DestinationFileGroup = req.MultipartForm.Value[uploadedFileOwnerGroupKey][0]
	}

	if len(req.MultipartForm.Value[uploadedFileModeKey]) > 0 {
		fileModeInt, err := strconv.ParseInt(req.MultipartForm.Value[uploadedFileModeKey][0], 8, 32)
		if err != nil {
			return errors2.Wrapf(err, "failed to parse file mode value %s", req.MultipartForm.Value[uploadedFileModeKey][0])
		}
		uf.DestinationFileMode = os.FileMode(fileModeInt)
	}

	if len(req.MultipartForm.Value[fileWriteForcedKey]) > 0 {
		uf.ForceWrite, err = strconv.ParseBool(req.MultipartForm.Value[fileWriteForcedKey][0])
		if err != nil {
			return err
		}
	}

	if len(req.MultipartForm.Value[fileSyncdKey]) > 0 {
		uf.Sync, err = strconv.ParseBool(req.MultipartForm.Value[fileSyncdKey][0])
		if err != nil {
			return err
		}
	}
	if len(req.MultipartForm.Value[IDKey]) > 0 {
		uf.ID = req.MultipartForm.Value[IDKey][0]
	}

	return nil
}

func (uf *UploadedFile) FromBytes(rawData []byte) error {
	return json.Unmarshal(rawData, uf)
}

func (uf *UploadedFile) ToBytes() (data []byte, err error) {
	return json.Marshal(uf)
}

type UploadResponse struct {
	UploadResponseShort
	Message string `json:"message"`
	Status  string `json:"status"`
}

type UploadResponseShort struct {
	ID        string `json:"uuid"`
	Filepath  string `json:"filepath"`
	SizeBytes int64  `json:"size"`
}
