package chserver

import (
	"fmt"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/models"
	"mime/multipart"
	"net/http"
	"path"

	"github.com/pkg/errors"

	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/clients"
)

const uploadBufSize = 1000000 // 1Mb
const defaultDirMode = 0764

type UploadResponse struct {
	Filepath  string `json:"filepath"`
	SizeBytes int64  `json:"size"`
}

type UploadRequest struct {
	File                 multipart.File
	FileHeader           *multipart.FileHeader
	ClientIDs            []string
	GroupIDs             []string
	clientsInGroupsCount int
	Clients              []*clients.Client
	models.UploadedFile
}

func (al *APIListener) handleFileUploads(w http.ResponseWriter, req *http.Request) {
	al.auditLog.Entry(auditlog.ApplicationFiles, auditlog.ActionCreate).
		WithHTTPRequest(req).
		Save()

	wasCreated, err := files.CreateDirIfNotExists(al.config.GetUploadDir(), defaultDirMode)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if wasCreated {
		al.Infof("created directory %s", al.config.GetUploadDir())
	}

	uploadRequest, err := al.uploadRequestFromRequest(req)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, err.Error())
		return
	}
	defer uploadRequest.File.Close()

	uploadRequest.SourceFilePath = path.Join(al.config.GetUploadDir(), uploadRequest.FileHeader.Filename)

	err = validateUploadRequest(uploadRequest)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, err.Error())
		return
	}

	al.Infof(
		"will upload file %s, size %d, Content-Type %s",
		uploadRequest.FileHeader.Filename,
		uploadRequest.FileHeader.Size,
		uploadRequest.FileHeader.Header.Get("Content-Type"),
	)

	copiedBytes, err := files.CopyFileToDestination(uploadRequest.SourceFilePath, uploadRequest.File, al.Logger)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.sendFileToClientsAsync(uploadRequest)

	al.writeJSONResponse(w, http.StatusOK, &UploadResponse{
		Filepath:  uploadRequest.DestinationPath,
		SizeBytes: copiedBytes,
	})
}

func (al *APIListener) sendFileToClientsAsync(uploadRequest *UploadRequest) {
	go func(ur *UploadRequest) {
		err := al.sendFileToClients(ur)
		if err != nil {
			//todo properly handle errors
			al.Errorf("failed to upload file to clients: %v", err)
			return
		}
	}(uploadRequest)
}

func (al *APIListener) sendFileToClients(uploadRequest *UploadRequest) error {
	requestBytes, err := uploadRequest.ToBytes()
	if err != nil {
		return err
	}
	for _, cl := range uploadRequest.Clients {
		isSuccess, resp, err := cl.Connection.SendRequest(comm.RequestTypeUpload, true, requestBytes)
		if err != nil {
			return fmt.Errorf("failed to upload file %s: %v", uploadRequest.FileHeader.Filename, err)
		}
		al.Infof("send upload request to client %s, resp: %v, %s", cl.ID, isSuccess, string(resp))
	}

	return nil
}

func (al *APIListener) uploadRequestFromRequest(req *http.Request) (*UploadRequest, error) {
	ur := &UploadRequest{}
	err := req.ParseMultipartForm(uploadBufSize)
	if err != nil {
		return nil, err
	}

	ur.ClientIDs = req.MultipartForm.Value["client"]
	ur.GroupIDs = req.MultipartForm.Value["group_id"]

	ur.Clients, ur.clientsInGroupsCount, err = al.getOrderedClients(req.Context(), ur.ClientIDs, ur.GroupIDs)
	if err != nil {
		return nil, err
	}

	err = ur.FromMultipartRequest(req)
	if err != nil {
		return nil, err
	}

	ur.File, ur.FileHeader, err = req.FormFile("upload")
	if err != nil {
		return nil, err
	}

	return ur, nil
}

func validateUploadRequest(ur *UploadRequest) error {
	if len(ur.ClientIDs) == 0 && ur.clientsInGroupsCount == 0 {
		return errors.New("at least 1 client should be specified")
	}

	if len(ur.GroupIDs) > 0 && ur.clientsInGroupsCount == 0 && len(ur.ClientIDs) == 0 {
		return errors.New("No active clients belong to the selected group(s)")
	}

	if len(ur.Clients) == 0 {
		return errors.New("no active clients found for the provided criteria")
	}

	return ur.Validate()
}
