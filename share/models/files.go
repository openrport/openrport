package models

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	errors2 "github.com/pkg/errors"
)

const (
	uploadedFileKey                = "upload"
	uploadedFileDestinationPathKey = "dest"
	uploadedFileOwnerKey           = "user"
	uploadedFileOwnerGroupKey      = "group"
	uploadedFileModeKey            = "mode"
	boundary                       = "313uidj" //some random predictable boundary for marking multipart items
)

type UploadedFile struct {
	File                 multipart.File
	FileHeader           *multipart.FileHeader
	TempFilePath         string
	DestinationPath      string
	DestinationFileMode  os.FileMode
	DestinationFileOwner string
	DestinationFileGroup string
	ForceWrite           bool
}

func (uf UploadedFile) Validate() error {
	if uf.FileHeader.Filename == "" {
		return errors.New("empty source file name")
	}

	if uf.DestinationPath == "" {
		return errors.New("empty destination file path")
	}

	return nil
}

func (uf *UploadedFile) FromMultipartRequest(req *http.Request) error {
	var err error
	uf.File, uf.FileHeader, err = req.FormFile(uploadedFileKey)
	if err != nil {
		return err
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

	return nil
}

func (uf *UploadedFile) FromMultipartData(rawData []byte) error {
	req, err := http.NewRequest("POST", "http://localhost/", bytes.NewReader(rawData))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "multipart/form-data; boundary="+boundary)

	return uf.FromMultipartRequest(req)
}

func (uf *UploadedFile) ToMultipartData() (data []byte, err error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	err = writer.SetBoundary(boundary)
	if err != nil {
		return nil, err
	}

	err = writer.WriteField(uploadedFileDestinationPathKey, uf.DestinationPath)
	if err != nil {
		return nil, err
	}

	if uf.DestinationFileOwner != "" {
		err = writer.WriteField(uploadedFileOwnerKey, uf.DestinationFileOwner)
		if err != nil {
			return nil, err
		}
	}

	if uf.DestinationFileGroup != "" {
		err = writer.WriteField(uploadedFileOwnerGroupKey, uf.DestinationFileGroup)
		if err != nil {
			return nil, err
		}
	}

	if uf.DestinationFileMode > 0 {
		err = writer.WriteField(uploadedFileModeKey, fmt.Sprintf("%04o", uf.DestinationFileMode))
		if err != nil {
			return nil, err
		}
	}

	fw, err := writer.CreateFormFile(uploadedFileKey, uf.FileHeader.Filename)
	if err != nil {
		return nil, err
	}

	if uf.File == nil && uf.TempFilePath != "" {
		uf.File, err = os.Open(uf.TempFilePath)
		if err != nil {
			return nil, err
		}
	}

	if uf.File != nil {
		_, err = io.Copy(fw, uf.File)
		if err != nil {
			return nil, err
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return body.Bytes(), nil
}
