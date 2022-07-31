package chserver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/validation"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

var (
	ErrRequestIncludesMultipleTargetingParams = errors.New("multiple targeting options are not supported. Please specify only one")
	ErrRequestMissingTargetingParams          = errors.New("please specify targeting options, such as client ids, groups ids or tags")
	ErrMissingTagsInMultiJobRequest           = errors.New("please specify tags in the tags list")
)

func (al *APIListener) handleCommandsExecutionWS(
	ctx context.Context,
	uiConnTS *ws.ConcurrentWebSocket,
	inboundMsg *jobs.MultiJobRequest,
	clientsInGroupsCount int,
	auditLogEntry *auditlog.Entry,
) {
	if inboundMsg.Command == "" {
		uiConnTS.WriteError("Command cannot be empty.", nil)
		return
	}
	if err := validation.ValidateInterpreter(inboundMsg.Interpreter, inboundMsg.IsScript); err != nil {
		uiConnTS.WriteError("Invalid interpreter", err)
		return
	}

	if inboundMsg.TimeoutSec <= 0 {
		inboundMsg.TimeoutSec = al.config.Server.RunRemoteCmdTimeoutSec
	}

	if !hasClientTags(inboundMsg) {
		errTitle := validateNonClientsTagTargeting(inboundMsg, clientsInGroupsCount)
		if errTitle != "" {
			uiConnTS.WriteError(errTitle, nil)
			return
		}
	} else {
		errTitle := validateClientTagsTargeting(inboundMsg)
		if errTitle != "" {
			uiConnTS.WriteError(errTitle, nil)
			return
		}
	}

	curUser, err := al.getUserModelForAuth(ctx)
	if err != nil {
		uiConnTS.WriteError("Could not get current user.", err)
		return
	}

	err = al.clientService.CheckClientsAccess(inboundMsg.OrderedClients, curUser)
	if err != nil {
		uiConnTS.WriteError(err.Error(), nil)
		return
	}

	jid, err := generateNewJobID()
	if err != nil {
		uiConnTS.WriteError("Could not generate job id.", err)
		return
	}
	al.Server.uiJobWebSockets.Set(jid, uiConnTS)
	defer al.Server.uiJobWebSockets.Delete(jid)

	auditLogEntry.
		WithRequest(inboundMsg).
		WithID(jid).
		SaveForMultipleClients(inboundMsg.OrderedClients)

	createdBy := curUser.Username
	if inboundMsg.OrderedClients != nil && len(inboundMsg.OrderedClients) > 0 {
		// by default abortOnErr is true
		abortOnErr := true
		if inboundMsg.AbortOnError != nil {
			abortOnErr = *inboundMsg.AbortOnError
		}

		multiJob := &models.MultiJob{
			MultiJobSummary: models.MultiJobSummary{
				JID:       jid,
				StartedAt: time.Now(),
				CreatedBy: createdBy,
			},
			ClientIDs:   inboundMsg.ClientIDs,
			GroupIDs:    inboundMsg.GroupIDs,
			ClientTags:  inboundMsg.ClientTags,
			Command:     inboundMsg.Command,
			Cwd:         inboundMsg.Cwd,
			Interpreter: inboundMsg.Interpreter,
			TimeoutSec:  inboundMsg.TimeoutSec,
			Concurrent:  inboundMsg.ExecuteConcurrently,
			AbortOnErr:  abortOnErr,
			IsSudo:      inboundMsg.IsSudo,
			IsScript:    inboundMsg.IsScript,
		}
		if err := al.jobProvider.SaveMultiJob(multiJob); err != nil {
			uiConnTS.WriteError("Failed to persist a new multi-client job.", err)
			return
		}

		al.Debugf("Multi-client Job[id=%q] created to execute remote command on clients %s, groups %s tags %s: %q.", multiJob.JID, inboundMsg.ClientIDs, inboundMsg.GroupIDs, getClientTags(inboundMsg), inboundMsg.Command)

		uiConnTS.SetWritesBeforeClose(len(inboundMsg.OrderedClients))

		// for sequential execution - create a channel to get the job result
		var curJobDoneChannel chan *models.Job

		if !multiJob.Concurrent {
			curJobDoneChannel = make(chan *models.Job)
			al.jobsDoneChannel.Set(multiJob.JID, curJobDoneChannel)
			defer func() {
				close(curJobDoneChannel)
				al.jobsDoneChannel.Del(multiJob.JID)
			}()
		}

		for _, client := range inboundMsg.OrderedClients {
			curJID, err := generateNewJobID()
			if err != nil {
				uiConnTS.WriteError("Could not generate job id.", err)
				return
			}
			if multiJob.Concurrent {
				go al.createAndRunJobWS(
					uiConnTS,
					&jid,
					curJID,
					inboundMsg.Command,
					multiJob.Interpreter,
					createdBy,
					multiJob.Cwd,
					multiJob.TimeoutSec,
					multiJob.IsSudo,
					multiJob.IsScript,
					client,
				)
			} else {
				success := al.createAndRunJobWS(
					uiConnTS,
					&jid,
					curJID,
					inboundMsg.Command,
					multiJob.Interpreter,
					createdBy,
					multiJob.Cwd,
					multiJob.TimeoutSec,
					multiJob.IsSudo,
					multiJob.IsScript,
					client,
				)

				if !success {
					if multiJob.AbortOnErr {
						uiConnTS.Close()
						return
					}
					continue
				}

				if al.insecureForTests {
					continue
				}

				// wait until command is finished
				jobResult := <-curJobDoneChannel
				if multiJob.AbortOnErr && jobResult.Status == models.JobStatusFailed {
					uiConnTS.Close()
					return
				}
			}
		}
	} else {
		client := inboundMsg.OrderedClients[0]
		al.createAndRunJobWS(
			uiConnTS,
			nil,
			jid,
			inboundMsg.Command,
			inboundMsg.Interpreter,
			createdBy,
			inboundMsg.Cwd,
			inboundMsg.TimeoutSec,
			inboundMsg.IsSudo,
			inboundMsg.IsScript,
			client,
		)
	}

	if al.testDone != nil {
		al.testDone <- true
	}

	// check for Close message from client to close the connection
	mt, message, err := uiConnTS.ReadMessage()
	if err != nil {
		if closeErr, ok := err.(*websocket.CloseError); ok {
			al.Debugf("Received a closed err on WS read: %v", closeErr)
			return
		}
		al.Debugf("Error read from websocket: %v", err)
		return
	}

	al.Debugf("Message received: type %v, msg %s", mt, message)
	uiConnTS.Close()
}

func checkTargetingParams(request *jobs.MultiJobRequest) (errTitle string, err error) {
	if request.ClientIDs == nil && request.GroupIDs == nil && request.ClientTags == nil {
		return "Missing targeting parameters.", ErrRequestMissingTargetingParams
	}
	if request.ClientIDs != nil && request.ClientTags != nil ||
		request.GroupIDs != nil && request.ClientTags != nil {
		return "Multiple targeting parameters.", ErrRequestIncludesMultipleTargetingParams
	}
	if request.ClientTags != nil {
		if request.ClientTags.Tags == nil || (request.ClientTags.Tags != nil && len(request.ClientTags.Tags) == 0) {
			return "No tags specified.", ErrMissingTagsInMultiJobRequest
		}
	}
	return "", nil
}

func validateNonClientsTagTargeting(request *jobs.MultiJobRequest, groupClientsCount int) (errTitle string) {
	if len(request.GroupIDs) > 0 && groupClientsCount == 0 && len(request.ClientIDs) == 0 {
		return "No active clients belong to the selected group(s)."
	}

	minClients := 2
	if len(request.ClientIDs) < minClients && groupClientsCount == 0 {
		return fmt.Sprintf("At least %d clients should be specified.", minClients)
	}

	return ""
}

func validateClientTagsTargeting(request *jobs.MultiJobRequest) (errTitle string) {
	minClients := 1
	if request.OrderedClients == nil || len(request.OrderedClients) < minClients {
		return fmt.Sprintf("At least %d client should be specified.", minClients)
	}
	return ""
}

func getClientTags(request *jobs.MultiJobRequest) (tags []string) {
	tags = []string{}
	if request.ClientTags != nil {
		tags = request.ClientTags.Tags
	}
	return tags
}

func hasClientTags(request *jobs.MultiJobRequest) (has bool) {
	return request.ClientTags != nil
}
