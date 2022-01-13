package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/cloudradar-monitoring/rport/share/logger"

	errors2 "github.com/pkg/errors"
)

const (
	uploadedFileDestinationPathKey = "dest"
	uploadedFileOwnerKey           = "user"
	uploadedFileOwnerGroupKey      = "group"
	uploadedFileModeKey            = "mode"
	fileWriteForcedKey             = "force"
	fileSyncdKey                   = "sync"
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
	destinationDir := path.Dir(uf.DestinationPath)
	for _, p := range globPatters {
		matched, err := path.Match(p, destinationDir)
		if err != nil {
			log.Errorf("failed to match glob pattern %s against file name %s: %v", p, uf.DestinationPath, err)
			continue
		}

		if matched {
			return fmt.Errorf("target path %s matches file_push_deny pattern %s, therefore the file push request is rejected", uf.DestinationPath, p)
		}
	}

	return nil
}

func (uf *UploadedFile) FromMultipartRequest(req *http.Request) error {
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
		uf.ForceWrite = StrToBool(req.MultipartForm.Value[fileWriteForcedKey][0])
	}
	if len(req.MultipartForm.Value[fileSyncdKey]) > 0 {
		uf.Sync = StrToBool(req.MultipartForm.Value[fileSyncdKey][0])
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
