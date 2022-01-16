package chserver

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const uploadBufSize = 1000000 // 1Mb

type UploadRequest struct {
	File                 multipart.File
	FileHeader           *multipart.FileHeader
	ClientIDs            []string
	GroupIDs             []string
	clientsInGroupsCount int
	Clients              []*clients.Client
	*models.UploadedFile
}

func (al *APIListener) handleFileUploads(w http.ResponseWriter, req *http.Request) {
	uploadRequest, err := al.uploadRequestFromRequest(req)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, err.Error())
		return
	}
	defer uploadRequest.File.Close()

	wasCreated, err := al.filesAPI.CreateDirIfNotExists(al.config.GetUploadDir(), files.DefaultMode)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if wasCreated {
		al.Infof("created directory %s", al.config.GetUploadDir())
	}

	uploadRequest.SourceFilePath = al.genFilePath(uploadRequest.ID)

	err = validateUploadRequest(uploadRequest)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, err.Error())
		return
	}

	copiedBytes, md5Checksum, err := al.filesAPI.CreateFile(uploadRequest.SourceFilePath, uploadRequest.File)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	uploadRequest.Md5Checksum = md5Checksum

	al.Debugf(
		"stored file %s on server, size %d, Content-Type %s, temp location: %s, md5 checksum: %x",
		uploadRequest.FileHeader.Filename,
		uploadRequest.FileHeader.Size,
		uploadRequest.FileHeader.Header.Get("Content-Type"),
		uploadRequest.SourceFilePath,
		md5Checksum,
	)

	resp := &models.UploadResponseShort{
		ID:        uploadRequest.ID,
		Filepath:  uploadRequest.DestinationPath,
		SizeBytes: copiedBytes,
	}
	al.auditLog.Entry(auditlog.ApplicationUploads, auditlog.ActionCreate).
		WithHTTPRequest(req).
		WithRequest(uploadRequest.UploadedFile).
		WithResponse(resp).
		WithID(uploadRequest.UploadedFile.ID).
		SaveForMultipleClients(uploadRequest.Clients)

	go al.sendFileToClients(uploadRequest)

	al.writeJSONResponse(w, http.StatusOK, resp)
}

func (al *APIListener) handleUploadsWS(w http.ResponseWriter, req *http.Request) {
	uiConn, err := apiUpgrader.Upgrade(w, req, nil)
	if err != nil {
		al.Errorf("Failed to establish WS connection: %v", err)
		return
	}

	connID, err := uuid.NewUUID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	al.Server.uploadWebSockets.Store(connID, uiConn)

	defer al.Server.uploadWebSockets.Delete(connID)
	defer uiConn.Close()

	for {
		_, _, err := uiConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				al.Infof("closed ws connection: %v", err)
			}
			break
		}
	}
}

func (al *APIListener) genFilePath(uuid string) string {
	uniqueFilename := fmt.Sprintf("%s_rport_filepush", uuid)

	return filepath.Join(al.config.GetUploadDir(), uniqueFilename)
}

type uploadResult struct {
	resp      *models.UploadResponse
	err       error
	isSuccess bool
	client    *clients.Client
}

type UploadOutput struct {
	ClientID string `json:"client_id"`
	*models.UploadResponse
}

func (al *APIListener) sendFileToClients(uploadRequest *UploadRequest) {
	requestBytes, err := uploadRequest.UploadedFile.ToBytes()
	if err != nil {
		wrappedErr := errors.Wrapf(err, "failed to convert upload request to bytes")
		output := &UploadOutput{
			UploadResponse: &models.UploadResponse{
				Message: wrappedErr.Error(),
				Status:  "error",
			},
		}

		al.notifyUploadEventListeners(output)
		al.Errorf(wrappedErr.Error())
		return
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(uploadRequest.Clients))

	resChan := make(chan *uploadResult, len(uploadRequest.Clients))

	for _, cl := range uploadRequest.Clients {
		go al.sendFileToClient(wg, requestBytes, cl, resChan)
	}

	go func() {
		wg.Wait()
		close(resChan)
	}()

	al.consumeUploadResults(resChan, uploadRequest)

	err = al.filesAPI.Remove(uploadRequest.SourceFilePath)
	if err != nil {
		al.Errorf("failed to delete temp file path %s: %v", uploadRequest.SourceFilePath, err)
	}
}

func (al *APIListener) consumeUploadResults(resChan chan *uploadResult, uploadRequest *UploadRequest) {
	for res := range resChan {
		output := &UploadOutput{
			ClientID:       res.client.ID,
			UploadResponse: res.resp,
		}
		if res.err != nil {
			output.UploadResponse = &models.UploadResponse{
				Message: res.err.Error(),
				Status:  "error",
				UploadResponseShort: models.UploadResponseShort{
					ID: uploadRequest.ID,
				},
			}
			al.Errorf(
				"upload failure: %v, file id: %s, file path: %s, client %s",
				res.err,
				uploadRequest.ID,
				uploadRequest.DestinationPath,
				res.client.ID,
			)
			al.auditLog.Entry(auditlog.ApplicationUploads, auditlog.ActionFailed).
				WithRequest(uploadRequest.UploadedFile).
				WithResponse(output).
				WithID(uploadRequest.UploadedFile.ID).
				WithClient(res.client).
				Save()
		} else {
			al.Infof(
				"upload success, file id: %s, file path: %s, client %s",
				uploadRequest.ID,
				uploadRequest.DestinationPath,
				res.client.ID,
			)
			al.auditLog.Entry(auditlog.ApplicationUploads, auditlog.ActionSuccess).
				WithRequest(uploadRequest.UploadedFile).
				WithResponse(output).
				WithID(uploadRequest.UploadedFile.ID).
				WithClient(res.client).
				Save()
		}

		al.notifyUploadEventListeners(output)
	}
}

func (al *APIListener) sendFileToClient(wg *sync.WaitGroup, requestBytes []byte, cl *clients.Client, resChan chan *uploadResult) {
	defer wg.Done()
	isSuccess, respBytes, err := cl.Connection.SendRequest(comm.RequestTypeUpload, true, requestBytes)
	if err != nil {
		resChan <- &uploadResult{
			err:       err,
			isSuccess: isSuccess,
			client:    cl,
		}
		return
	}
	if !isSuccess {
		resChan <- &uploadResult{
			err:       errors.New(string(respBytes)),
			isSuccess: false,
			client:    cl,
		}
		return
	}

	resp := &models.UploadResponse{}
	err = json.Unmarshal(respBytes, resp)
	if err != nil {
		resChan <- &uploadResult{
			err:       errors.Wrapf(err, "failed to parse %s", string(respBytes)),
			isSuccess: isSuccess,
			client:    cl,
		}
		return
	}

	resChan <- &uploadResult{
		isSuccess: isSuccess,
		client:    cl,
		resp:      resp,
	}
}

func (al *APIListener) notifyUploadEventListeners(msg interface{}) {
	al.uploadWebSockets.Range(func(key, value interface{}) bool {
		if wsConn, ok := value.(*websocket.Conn); ok {
			err := wsConn.WriteJSON(msg)
			if err != nil {
				al.Errorf("failed to send notification to websocket client %s: %v", key, err)
			}
		}
		return true
	})
}

func (al *APIListener) uploadRequestFromRequest(req *http.Request) (*UploadRequest, error) {
	ur := &UploadRequest{
		UploadedFile: &models.UploadedFile{},
	}

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

	if ur.UploadedFile.ID == "" {
		id, e := random.UUID4()
		if e != nil {
			al.Errorf("failed to generate uuid, will fallback to timestamp uuid, error: %v", e)
			id = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		ur.UploadedFile.ID = id
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
