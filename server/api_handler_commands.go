package chserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/validation"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

type jobPayload struct {
	JID         *string     `json:"jid,omitempty"`
	Status      *string     `json:"status,omitempty"`
	FinishedAt  **time.Time `json:"finished_at,omitempty"`
	ClientID    *string     `json:"client_id,omitempty"`
	ClientName  *string     `json:"client_name,omitempty"`
	Command     *string     `json:"command,omitempty"`
	Cwd         *string     `json:"cwd,omitempty"`
	Interpreter *string     `json:"interpreter,omitempty"`
	PID         **int       `json:"pid,omitempty"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CreatedBy   *string     `json:"created_by,omitempty"`
	TimeoutSec  *int        `json:"timeout_sec,omitempty"`
	MultiJobID  **string    `json:"multi_job_id,omitempty"`
	ScheduleID  **string    `json:"schedule_id,omitempty"`
	Error       *string     `json:"error,omitempty"`
	Result      **jobResult `json:"result,omitempty"`
	IsSudo      *bool       `json:"is_sudo,omitempty"`
	IsScript    *bool       `json:"is_script,omitempty"`
}

type jobResult struct {
	StdOut  *string `json:"stdout,omitempty"`
	StdErr  *string `json:"stderr,omitempty"`
	Summary *string `json:"summary,omitempty"`
}

func convertToJobsPayload(jobs []*models.Job, fields []query.FieldsOption) []jobPayload {
	requestedFields := query.RequestedFields(fields, "jobs")
	if len(requestedFields) == 0 {
		requestedFields = query.RequestedFields(fields, "commands")
		if len(requestedFields) == 0 {
			requestedFields = query.RequestedFields(fields, "scripts")
		}
	}
	requestedResultFields := query.RequestedFields(fields, "result")

	result := make([]jobPayload, len(jobs))
	for i, job := range jobs {
		if requestedFields["jid"] {
			result[i].JID = &job.JID
		}
		if requestedFields["status"] {
			result[i].Status = &job.Status
		}
		if requestedFields["finished_at"] {
			result[i].FinishedAt = &job.FinishedAt
		}
		if requestedFields["client_id"] {
			result[i].ClientID = &job.ClientID
		}
		if requestedFields["client_name"] {
			result[i].ClientName = &job.ClientName
		}
		if requestedFields["command"] {
			result[i].Command = &job.Command
		}
		if requestedFields["cwd"] {
			result[i].Cwd = &job.Cwd
		}
		if requestedFields["interpreter"] {
			result[i].Interpreter = &job.Interpreter
		}
		if requestedFields["pid"] {
			result[i].PID = &job.PID
		}
		if requestedFields["started_at"] {
			result[i].StartedAt = &job.StartedAt
		}
		if requestedFields["created_by"] {
			result[i].CreatedBy = &job.CreatedBy
		}
		if requestedFields["timeout_sec"] {
			result[i].TimeoutSec = &job.TimeoutSec
		}
		if requestedFields["multi_job_id"] {
			result[i].MultiJobID = &job.MultiJobID
		}
		if requestedFields["schedule_id"] {
			result[i].ScheduleID = &job.ScheduleID
		}
		if requestedFields["error"] {
			result[i].Error = &job.Error
		}
		if requestedFields["is_sudo"] {
			result[i].IsSudo = &job.IsSudo
		}
		if requestedFields["is_script"] {
			result[i].IsScript = &job.IsScript
		}
		if len(requestedResultFields) > 0 {
			result[i].Result = new(*jobResult)
			if job.Result != nil {
				(*result[i].Result) = &jobResult{}
				if requestedResultFields["stdout"] {
					(*result[i].Result).StdOut = &job.Result.StdOut
				}
				if requestedResultFields["stderr"] {
					(*result[i].Result).StdErr = &job.Result.StdErr
				}
				if requestedResultFields["summary"] {
					(*result[i].Result).Summary = &job.Result.Summary
				}
			}
		}
	}

	return result
}

type newJobResponse struct {
	JID string `json:"jid"`
}

// handlePostCommand handles POST /clients/{client_id}/commands
func (al *APIListener) handlePostCommand(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cid := vars[routeParamClientID]
	if cid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamClientID))
		return
	}

	execCmdInput := &api.ExecuteInput{}
	err := parseRequestBody(req.Body, &execCmdInput)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	execCmdInput.ClientID = cid
	execCmdInput.IsScript = false

	resp := al.handleExecuteCommand(req.Context(), w, execCmdInput)

	if resp != nil {
		al.auditLog.Entry(auditlog.ApplicationClientCommand, auditlog.ActionExecuteStart).
			WithHTTPRequest(req).
			WithClientID(cid).
			WithRequest(execCmdInput).
			WithResponse(resp).
			WithID(resp.JID).
			Save()
	}
}

// handleGetCommands handles GET /clients/{client_id}/commands
func (al *APIListener) handleGetCommands(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cid := vars[routeParamClientID]
	if cid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamClientID))
		return
	}

	options := query.NewOptions(req, nil, nil, jobs.JobListDefaultFields)

	err := query.ValidateListOptions(options, jobs.JobSupportedSorts, jobs.JobSupportedFilters, jobs.JobSupportedFields, &query.PaginationConfig{
		MaxLimit:     jobs.MaxLimit,
		DefaultLimit: jobs.DefaultLimit,
	})
	if err != nil {
		al.jsonError(w, err)
		return
	}

	options.Filters = append(options.Filters, query.FilterOption{Column: []string{"client_id"}, Values: []string{cid}})
	result, err := al.jobProvider.List(req.Context(), options)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get client jobs: client_id=%q.", cid), err)
		return
	}

	totalCount, err := al.jobProvider.Count(req.Context(), options)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get client jobs: client_id=%q.", cid), err)
		return
	}

	payload := &api.SuccessPayload{
		Data: convertToJobsPayload(result, options.Fields),
		Meta: api.NewMeta(totalCount),
	}
	al.writeJSONResponse(w, http.StatusOK, payload)
}

// handleGetMultiClientCommandJobs handles GET /commands/{job_id}/jobs
func (al *APIListener) handleGetMultiClientCommandJobs(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	multiJobID := vars[routeParamJobID]
	if multiJobID == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamJobID))
		return
	}

	options := query.NewOptions(req, nil, nil, jobs.JobListDefaultFields)

	err := query.ValidateListOptions(options, jobs.JobSupportedSorts, jobs.JobSupportedFilters, jobs.JobSupportedFields, &query.PaginationConfig{
		MaxLimit:     jobs.MaxLimit,
		DefaultLimit: jobs.DefaultLimit,
	})
	if err != nil {
		al.jsonError(w, err)
		return
	}

	options.Filters = append(options.Filters, query.FilterOption{Column: []string{"multi_job_id"}, Values: []string{multiJobID}})
	result, err := al.jobProvider.List(req.Context(), options)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get jobs: multi_job_id=%q.", multiJobID), err)
		return
	}

	totalCount, err := al.jobProvider.Count(req.Context(), options)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get jobs: multi_job_id=%q.", multiJobID), err)
		return
	}

	payload := &api.SuccessPayload{
		Data: convertToJobsPayload(result, options.Fields),
		Meta: api.NewMeta(totalCount),
	}
	al.writeJSONResponse(w, http.StatusOK, payload)
}

// handleGetCommand handles GET /clients/{client_id}/commands/{job_id}
func (al *APIListener) handleGetCommand(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cid := vars[routeParamClientID]
	if cid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamClientID))
		return
	}
	jid := vars[routeParamJobID]
	if jid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamJobID))
		return
	}

	job, err := al.jobProvider.GetByJID(cid, jid)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to find a job[id=%q].", jid), err)
		return
	}
	if job == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Job[id=%q] not found.", jid))
		return
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(job))
}

// TODO: refactor to reuse similar code for REST API and WebSocket to execute cmds if both will be supported
// handlePostMultiClientCommand handles POST /commands
func (al *APIListener) handlePostMultiClientCommand(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var reqBody jobs.MultiJobRequest
	err := parseRequestBody(req.Body, &reqBody)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	errTitle, err := checkTargetingParams(&reqBody)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, errTitle, err)
		return
	}

	if reqBody.Command == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Command cannot be empty.")
		return
	}
	if err := validation.ValidateInterpreter(reqBody.Interpreter, reqBody.IsScript); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "Invalid interpreter.", err)
		return
	}

	var orderedClients []*clients.Client
	var groupClientsCount int

	if !hasClientTags(&reqBody) {
		// do the original client ids flow
		orderedClients, groupClientsCount, err = al.getOrderedClients(ctx, reqBody.ClientIDs, reqBody.GroupIDs, false /* allowDisconnected */)
		if err != nil {
			al.jsonError(w, err)
			return
		}
		reqBody.OrderedClients = orderedClients

		errTitle := validateNonClientsTagTargeting(&reqBody, groupClientsCount)
		if errTitle != "" {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, errTitle)
			return
		}
	} else {
		// do tags
		orderedClients, err = al.getOrderedClientsByTag(ctx, reqBody.ClientIDs, reqBody.GroupIDs, reqBody.ClientTags, false /* allowDisconnected */)
		if err != nil {
			al.jsonError(w, err)
			return
		}
		reqBody.OrderedClients = orderedClients

		errTitle := validateClientTagsTargeting(orderedClients)
		if errTitle != "" {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, errTitle)
			return
		}
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.clientService.CheckClientsAccess(reqBody.OrderedClients, curUser)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	reqBody.Username = curUser.Username

	multiJob, err := al.StartMultiClientJob(ctx, &reqBody)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	resp := newJobResponse{
		JID: multiJob.JID,
	}

	al.auditLog.Entry(auditlog.ApplicationClientCommand, auditlog.ActionExecuteStart).
		WithHTTPRequest(req).
		WithRequest(reqBody).
		WithResponse(resp).
		WithID(multiJob.JID).
		SaveForMultipleClients(reqBody.OrderedClients)

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(resp))

	al.Debugf("Multi-client Job[id=%q] created to execute remote command on clients %s, groups %s, tags %s: %q.", multiJob.JID, reqBody.ClientIDs, reqBody.GroupIDs, reqBody.GetTags(), reqBody.Command)
}

func (al *APIListener) handleExecuteCommand(ctx context.Context, w http.ResponseWriter, executeInput *api.ExecuteInput) *newJobResponse {
	if executeInput.Command == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Command cannot be empty.")
		return nil
	}
	if err := validation.ValidateInterpreter(executeInput.Interpreter, executeInput.IsScript); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusBadRequest, "Invalid interpreter.", err)
		return nil
	}

	if executeInput.TimeoutSec <= 0 {
		executeInput.TimeoutSec = al.config.Server.RunRemoteCmdTimeoutSec
	}

	client, err := al.clientService.GetActiveByID(executeInput.ClientID)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to find an active client with id=%q.", executeInput.ClientID), err)
		return nil
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Active client with id=%q not found.", executeInput.ClientID))
		return nil
	}

	// send the command to the client
	// Send a job with all possible info in order to get the full-populated job back (in client-listener) when it's done.
	// Needed when server restarts to get all job data from client. Because on server restart job running info is lost.
	jid, err := generateNewJobID()
	if err != nil {
		al.jsonError(w, err)
		return nil
	}
	curJob := models.Job{
		JID:         jid,
		FinishedAt:  nil,
		ClientID:    executeInput.ClientID,
		ClientName:  client.Name,
		Command:     executeInput.Command,
		Interpreter: executeInput.Interpreter,
		CreatedBy:   api.GetUser(ctx, al.Logger),
		TimeoutSec:  executeInput.TimeoutSec,
		Result:      nil,
		Cwd:         executeInput.Cwd,
		IsSudo:      executeInput.IsSudo,
		IsScript:    executeInput.IsScript,
	}
	sshResp := &comm.RunCmdResponse{}
	err = comm.SendRequestAndGetResponse(client.Connection, comm.RequestTypeRunCmd, curJob, sshResp)
	if err != nil {
		if _, ok := err.(*comm.ClientError); ok {
			al.jsonErrorResponseWithTitle(w, http.StatusConflict, err.Error())
		} else {
			al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to execute remote command.", err)
		}
		return nil
	}

	// set fields received in response
	curJob.PID = &sshResp.Pid
	curJob.StartedAt = sshResp.StartedAt
	curJob.Status = models.JobStatusRunning

	if err := al.jobProvider.CreateJob(&curJob); err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to persist a new job.", err)
		return nil
	}

	resp := &newJobResponse{
		JID: curJob.JID,
	}

	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(resp))

	al.Debugf("Job[id=%q] created to execute remote command on client with id=%q: %q.", curJob.JID, executeInput.ClientID, executeInput.Command)

	return resp
}

// handleCommandsWS handles GET /ws/commands
func (al *APIListener) handleCommandsWS(w http.ResponseWriter, req *http.Request) {
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
	} else if err != nil {
		uiConnTS.WriteError("Invalid JSON data.", err)
		return
	}

	errTitle, err := checkTargetingParams(inboundMsg)
	if err != nil {
		uiConnTS.WriteError(errTitle, err)
		return
	}

	var orderedClients []*clients.Client
	var clientsInGroupsCount int

	if !hasClientTags(inboundMsg) {
		// do the original client ids flow
		orderedClients, clientsInGroupsCount, err = al.getOrderedClients(ctx, inboundMsg.ClientIDs, inboundMsg.GroupIDs, false /* allowDisconnected */)
		if err != nil {
			uiConnTS.WriteError("", err)
			return
		}
	} else {
		// do tags
		orderedClients, err = al.getOrderedClientsByTag(ctx, inboundMsg.ClientIDs, inboundMsg.GroupIDs, inboundMsg.ClientTags, false /* allowDisconnected */)
		if err != nil {
			uiConnTS.WriteError("", err)
			return
		}

	}
	inboundMsg.OrderedClients = orderedClients
	inboundMsg.IsScript = false

	auditLogEntry := al.auditLog.Entry(auditlog.ApplicationClientCommand, auditlog.ActionExecuteStart).WithHTTPRequest(req)

	al.handleCommandsExecutionWS(ctx, uiConnTS, inboundMsg, clientsInGroupsCount, auditLogEntry)
}

// handleGetMultiClientCommand handles GET /commands/{job_id}
func (al *APIListener) handleGetMultiClientCommand(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	jid := vars[routeParamJobID]
	if jid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routeParamJobID))
		return
	}

	job, err := al.jobProvider.GetMultiJob(req.Context(), jid)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to find a multi-client job[id=%q].", jid), err)
		return
	}
	if job == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("Multi-client Job[id=%q] not found.", jid))
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if curUser.IsAdmin() || job.CreatedBy == curUser.Username {
		al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(job))
		return
	}
	al.jsonErrorResponseWithError(w, http.StatusForbidden, "forbidden", fmt.Errorf("you are not allowed to access items created by another user"))
}

// handleGetMultiClientCommands handles GET /commands
func (al *APIListener) handleGetMultiClientCommands(w http.ResponseWriter, req *http.Request) {
	listOptions := query.GetListOptions(req)

	err := query.ValidateListOptions(listOptions, jobs.MultiJobSupportedSorts, jobs.MultiJobSupportedFilters, nil /*fields*/, &query.PaginationConfig{
		MaxLimit:     1000,
		DefaultLimit: 100,
	})
	if err != nil {
		al.jsonError(w, err)
		return
	}

	result, err := al.jobProvider.GetMultiJobSummaries(req.Context(), listOptions)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to get multi-client jobs.", err)
		return
	}

	totalCount, err := al.jobProvider.CountMultiJobs(req.Context(), listOptions)
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to count multi-client jobs.", err)
		return
	}

	payload := &api.SuccessPayload{
		Data: result,
		Meta: api.NewMeta(totalCount),
	}
	al.writeJSONResponse(w, http.StatusOK, payload)
}
