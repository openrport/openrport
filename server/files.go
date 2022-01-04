package chserver

import (
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/pkg/errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
)

const uploadBufSize = 1000000 // 1Mb

type UploadResponse struct {
	Filepath  string `json:"filepath"`
	SizeBytes int64  `json:"size"`
}

type UploadRequest struct {
	ClientIDs            []string
	GroupIDs             []string
	clientsInGroupsCount int
	Clients              []*clients.Client
	File                 multipart.File
	FileHeader           *multipart.FileHeader
}

func (al *APIListener) handleFileUploads(w http.ResponseWriter, req *http.Request) {
	al.auditLog.Entry(auditlog.ApplicationFiles, auditlog.ActionCreate).
		WithHTTPRequest(req).
		Save()

	uploadFolder, err := prepareUploadFolder(al.config.Server.DataDir)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	al.Infof("will upload file to %s", uploadFolder)

	uploadRequest, err := al.uploadRequestFromRequest(req)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	defer uploadRequest.File.Close()

	err = validateUploadRequest(uploadRequest)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, err.Error())
		return
	}

	al.Infof(
		"file to upload %s, size %d, Content-Type %s",
		uploadRequest.FileHeader.Filename,
		uploadRequest.FileHeader.Size,
		uploadRequest.FileHeader.Header.Get("Content-Type"),
	)

	targetFilePath := path.Join(uploadFolder, uploadRequest.FileHeader.Filename)

	al.Infof("target file path: %s", targetFilePath)
	targetFile, err := os.OpenFile(targetFilePath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	defer targetFile.Close()

	copiedBytes, err := io.Copy(targetFile, uploadRequest.File)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	al.Infof("copied %d bytes to: %s", copiedBytes, targetFilePath)

	al.writeJSONResponse(w, http.StatusOK, &UploadResponse{
		Filepath:  targetFilePath,
		SizeBytes: copiedBytes,
	})
}

func (al *APIListener) uploadRequestFromRequest(req *http.Request) (*UploadRequest, error) {
	ur := &UploadRequest{}
	err := req.ParseMultipartForm(uploadBufSize)
	if err != nil {
		return nil, err
	}

	ur.ClientIDs = req.MultipartForm.Value["client"]
	ur.GroupIDs = req.MultipartForm.Value["group_id"]

	ur.File, ur.FileHeader, err = req.FormFile("upload")
	if err != nil {
		return nil, err
	}

	ur.Clients, ur.clientsInGroupsCount, err = al.getOrderedClients(req.Context(), ur.ClientIDs, ur.GroupIDs)
	if err != nil {
		return nil, err
	}

	return ur, nil
}

func validateUploadRequest(ur *UploadRequest) error {
	if len(ur.ClientIDs) == 0 && ur.clientsInGroupsCount == 0 {
		return errors.New("At least 1 client should be specified.")
	}

	if len(ur.GroupIDs) > 0 && ur.clientsInGroupsCount == 0 && len(ur.ClientIDs) == 0 {
		return errors.New("No active clients belong to the selected group(s).")
	}

	if len(ur.Clients) == 0 {
		return errors.New("no active clients found for the provided criteria")
	}

	return nil
}

func prepareUploadFolder(rootDir string) (string, error) {
	fileUploadFolder := path.Join(rootDir, "filepush")

	_, err := os.Stat(fileUploadFolder)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrapf(err, "failed to read folder info %s", fileUploadFolder)
		}

		err := os.MkdirAll(fileUploadFolder, 0764)
		if err != nil {
			return "", errors.Wrapf(err, "failed to create folder %s", fileUploadFolder)
		}
	}

	return fileUploadFolder, nil
}
