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

func (al *APIListener) handleGetClientAttributes(w http.ResponseWriter, req *http.Request) {

	ctx := req.Context()

	maybeClient := ctx.Value("client")
	if maybeClient == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusInternalServerError, fmt.Sprintf("client not present in the request"))
		return
	}
	client, ok := maybeClient.(*clients.Client)
	if !ok {
		al.jsonErrorResponseWithTitle(w, http.StatusInternalServerError, fmt.Sprintf("client is not of the client type"))
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(client.GetAttributes()))
}

type Resp struct {
	OK string `json:"ok"`
}

func (al *APIListener) handleUpdateClientAttributes(w http.ResponseWriter, req *http.Request) {

	ctx := req.Context()

	maybeClient := ctx.Value("client")
	if maybeClient == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusInternalServerError, fmt.Sprintf("client not present in the request"))
		return
	}
	client, ok := maybeClient.(*clients.Client)
	if !ok {
		al.jsonErrorResponseWithTitle(w, http.StatusInternalServerError, fmt.Sprintf("client is not of the client type"))
	}

	if req.ContentLength > 2^10*5 { // limit JSON to 5KB
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("request too big"))
		return
	}

	attributesRaw, err := io.ReadAll(req.Body)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("failed reading request: %v", err))
		return
	}

	attributes := clients.Attributes{}
	err = json.Unmarshal(attributesRaw, &attributes)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("failed parsing attributes: %v", err))
		return
	}

	sshResp := &Resp{}
	err = comm.SendRequestAndGetResponse(client.GetConnection(), comm.RequestTypeUpdateClientAttributes, attributes, sshResp, al.Log())
	if err != nil {
		if _, ok := err.(*comm.ClientError); ok {
			al.jsonErrorResponseWithTitle(w, http.StatusConflict, err.Error())
		} else {
			al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to execute remote command.", err)
		}
		return
	}

	client.SetAttributes(attributes)

	err = al.clientService.GetRepo().Save(client)
	if err != nil {
		al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload("client attributes updated, error saving changes to local db, changes will be visible after next client connection"))
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload("ok"))
}
