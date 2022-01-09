package models

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"

	errors2 "github.com/pkg/errors"
)

const (
	uploadedFileDestinationPathKey = "dest"
	uploadedFileOwnerKey           = "user"
	uploadedFileOwnerGroupKey      = "group"
	uploadedFileModeKey            = "mode"
)

type UploadedFile struct {
	SourceFilePath       string
	DestinationPath      string
	DestinationFileMode  os.FileMode
	DestinationFileOwner string
	DestinationFileGroup string
	ForceWrite           bool
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

	return nil
}

func (uf *UploadedFile) FromBytes(rawData []byte) error {
	return json.Unmarshal(rawData, uf)
}

func (uf *UploadedFile) ToBytes() (data []byte, err error) {
	return json.Marshal(uf)
}
