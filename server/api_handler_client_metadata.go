package chserver

import (
	"encoding/json"
	"fmt"
	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/clients"
	"github.com/realvnc-labs/rport/share/comm"
	"io"
	"net/http"
)

var copier = NewDataCopier(&clients.Client{}, &clients.Metadata{})

func NewDataCopier(from *clients.Client, to *clients.Metadata) func(from *clients.Client, to *clients.Metadata) {
	return func(from *clients.Client, to *clients.Metadata) {
		to.Tags = from.Tags
		to.Labels = from.Labels
	}
}

func (al *APIListener) handleGetClientMetadata(w http.ResponseWriter, req *http.Request) {

	ctx := req.Context()

	client := ctx.Value("client")
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusInternalServerError, fmt.Sprintf("client not present in the request"))
		return
	}
	client2 := client.(*clients.Client)

	resp := &clients.Metadata{}
	copier(client2, resp)

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(resp))
}

type Resp struct {
	OK string `json:"ok"`
}

func (al *APIListener) handleUpdateClientMetadata(w http.ResponseWriter, req *http.Request) {

	ctx := req.Context()

	client := ctx.Value("client")
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusInternalServerError, fmt.Sprintf("client not present in the request"))
		return
	}
	client2 := client.(*clients.Client)

	if req.ContentLength > 2^10*5 { // limit JSON to 5KB
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("too big request"))
		return
	}

	metadataRaw, err := io.ReadAll(req.Body)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("failed reading request: %v", err))
		return
	}

	metadata := &clients.Metadata{}
	err = json.Unmarshal(metadataRaw, metadata)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("failed parsing metadata: %v", err))
		return
	}

	sshResp := &Resp{}
	err = comm.SendRequestAndGetResponse(client2.GetConnection(), comm.RequestTypeUpdateClientMetadata, metadata, sshResp, al.Log())
	if err != nil {
		if _, ok := err.(*comm.ClientError); ok {
			al.jsonErrorResponseWithTitle(w, http.StatusConflict, err.Error())
		} else {
			al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to execute remote command.", err)
		}
		return
	}

	client2.SetMetadata(metadata)

	err = al.clientService.GetRepo().Save(client2)
	if err != nil {
		al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload("client metadata updated, error saving changes to local db"))
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload("ok"))
}
