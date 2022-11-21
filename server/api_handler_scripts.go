package chserver

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cloudradar-monitoring/rport/server/api"
	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/routes"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

// handleExecuteScript handles GET /clients/{client_id}/scripts
func (al *APIListener) handleExecuteScript(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cid := vars[routes.ParamClientID]
	if cid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routes.ParamClientID))
		return
	}

	execCmdInput := &api.ExecuteInput{}
	err := parseRequestBody(req.Body, &execCmdInput)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if execCmdInput.Script == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Missing script body")
		return
	}

	decodedScriptBytes, err := base64.StdEncoding.DecodeString(execCmdInput.Script)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusBadRequest, err)
		return
	}
	execCmdInput.Command = string(decodedScriptBytes)

	execCmdInput.ClientID = cid
	execCmdInput.IsScript = true

	resp := al.handleExecuteCommand(req.Context(), w, execCmdInput)

	if resp != nil {
		al.auditLog.Entry(auditlog.ApplicationClientScript, auditlog.ActionExecuteStart).
			WithHTTPRequest(req).
			WithClientID(cid).
			WithRequest(execCmdInput).
			WithResponse(resp).
			WithID(resp.JID).
			Save()
	}
}

// handlePostMultiClientScript handles POST /scripts
func (al *APIListener) handlePostMultiClientScript(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	inboundMsg := new(jobs.MultiJobRequest)
	err := parseRequestBody(req.Body, inboundMsg)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	orderedClients, _, err := al.getOrderedClientsWithValidation(ctx, inboundMsg)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	inboundMsg.OrderedClients = orderedClients

	err = al.enrichScriptInput(inboundMsg)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}
	clientGroups, err := al.clientGroupProvider.GetAll(ctx)
	if err != nil {
		al.jsonError(w, err)
	}
	err = al.clientService.CheckClientsAccess(inboundMsg.OrderedClients, curUser, clientGroups)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	inboundMsg.Username = curUser.Username

	multiJob, err := al.StartMultiClientJob(ctx, inboundMsg)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	resp := newJobResponse{
		JID: multiJob.JID,
	}

	al.auditLog.Entry(auditlog.ApplicationClientScript, auditlog.ActionExecuteStart).
		WithHTTPRequest(req).
		WithRequest(inboundMsg).
		WithResponse(resp).
		WithID(multiJob.JID).
		SaveForMultipleClients(inboundMsg.OrderedClients)

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(resp))

	al.Debugf("Multi-client Job[id=%q] created to execute remote command on clients %s, groups %s, tags %s: %q.", multiJob.JID, inboundMsg.ClientIDs, inboundMsg.GroupIDs, inboundMsg.GetClientTags(), inboundMsg.Command)
}

// handleScriptsWS handles GET /ws/scripts
func (al *APIListener) handleScriptsWS(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	uiConn, err := apiUpgrader.Upgrade(w, req, nil)
	if err != nil {
		al.Errorf("Failed to establish WS connection: %v", err)
		return
	}

	uiConnTS := ws.NewConcurrentWebSocket(uiConn, al.Logger)

	inboundMsg := &jobs.MultiJobRequest{}
	err = uiConnTS.ReadJSON(inboundMsg)
	if err == io.EOF { // is handled separately to return an informative error message
		uiConnTS.WriteError("Inbound message should contain non empty json object with command data.", nil)
		return
	}
	if err != nil {
		uiConnTS.WriteError("Invalid JSON data.", err)
		return
	}

	orderedClients, _, responseErr := al.getOrderedClientsWithValidation(ctx, inboundMsg)
	if responseErr != nil {
		uiConnTS.WriteError("", responseErr)
		return
	}

	inboundMsg.OrderedClients = orderedClients

	err = al.enrichScriptInput(inboundMsg)
	if err != nil {
		uiConnTS.WriteError("Failed to create script on multiple clients", err)
		return
	}

	auditLogEntry := al.auditLog.Entry(auditlog.ApplicationClientScript, auditlog.ActionExecuteStart).WithHTTPRequest(req)

	al.handleCommandsExecutionWS(ctx, uiConnTS, inboundMsg, auditLogEntry)
}

func (al *APIListener) enrichScriptInput(
	inboundMsg *jobs.MultiJobRequest,
) (err error) {
	if inboundMsg.Script == "" {
		return errors2.APIError{
			Message:    "Missing script body",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	if inboundMsg.TimeoutSec <= 0 {
		inboundMsg.TimeoutSec = al.config.Server.RunRemoteCmdTimeoutSec
	}

	decodedScriptBytes, err := base64.StdEncoding.DecodeString(inboundMsg.Script)
	if err != nil {
		return errors2.APIError{
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
			Message:    "failed to decode script payload from base64",
		}
	}

	inboundMsg.Command = string(decodedScriptBytes)
	inboundMsg.IsScript = true

	return nil
}
