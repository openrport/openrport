package chserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/models"
)

var (
	ErrRequestIncludesMultipleTargetingParams = errors.New("multiple targeting options are not supported. Please specify only one")
	ErrRequestMissingTargetingParams          = errors.New("please specify targeting options, such as client ids, groups ids or tags")
	ErrMissingTagsInMultiJobRequest           = errors.New("please specify tags in the tags list")
)

type TargetingParams interface {
	GetClientIDs() (ids []string)
	GetGroupIDs() (ids []string)
	GetClientTags() (clientTags *models.JobClientTags)
	GetTags() (tags []string)
}

func (al *APIListener) getOrderedClientsWithValidation(
	ctx context.Context,
	params TargetingParams,
	minClients int,
) (targetedClients []*clients.Client, groupClientsCount int, isBadRequest bool, errTitle string, err error) {
	errTitle, err = checkTargetingParams(params)
	if err != nil {
		return nil, 0, true, errTitle, err
	}

	if !hasClientTags(params) {
		// do the original client ids flow
		targetedClients, groupClientsCount, err = al.getOrderedClients(ctx, params.GetClientIDs(), params.GetGroupIDs(), false /* allowDisconnected */)
		if err != nil {
			return nil, 0, false, "", err
		}

		err := validateNonClientsTagTargeting(params, groupClientsCount, targetedClients, minClients)
		if err != nil {
			return nil, 0, true, "", err
		}
	} else {
		// do tags
		targetedClients, err = al.getOrderedClientsByTag(params.GetClientTags(), false /* allowDisconnected */)
		if err != nil {
			return nil, 0, false, "", err
		}

		err := validateClientTagsTargeting(targetedClients)
		if err != nil {
			return nil, 0, true, "", err
		}
	}
	return targetedClients, groupClientsCount, false, "", nil
}

func (al *APIListener) getOrderedClients(
	ctx context.Context,
	clientIDs, groupIDs []string,
	allowDisconnected bool,
) (
	orderedClients []*clients.Client,
	groupClientsFoundCount int,
	err error,
) {
	groupClients, err := al.makeGroupClientsList(ctx, groupIDs)
	if err != nil {
		return nil, 0, err
	}
	groupClientsFoundCount = len(groupClients)

	orderedClients, usedClientIDs, err := al.makeClientsList(clientIDs, allowDisconnected)
	if err != nil {
		return orderedClients, groupClientsFoundCount, err
	}

	// append group clients
	for _, groupClient := range groupClients {
		if !usedClientIDs[groupClient.ID] {
			usedClientIDs[groupClient.ID] = true
			orderedClients = append(orderedClients, groupClient)
		}
	}

	return orderedClients, groupClientsFoundCount, nil
}

func (al *APIListener) makeGroupClientsList(ctx context.Context, groupIDs []string) (groupClients []*clients.Client, err error) {
	var groups []*cgroups.ClientGroup
	for _, groupID := range groupIDs {
		group, err := al.clientGroupProvider.Get(ctx, groupID)
		if err != nil {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Failed to get a client group with id=%q.", groupID),
				Err:        err,
				HTTPStatus: http.StatusInternalServerError,
			}
			return nil, err
		}
		if group == nil {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Unknown group with id=%q.", groupID),
				Err:        err,
				HTTPStatus: http.StatusBadRequest,
			}
			return nil, err
		}
		groups = append(groups, group)
	}
	groupClients = al.clientService.GetActiveByGroups(groups)
	return groupClients, nil
}

func (al *APIListener) makeClientsList(clientIDs []string, allowDisconnected bool) (orderedClients []*clients.Client, usedClientIDs map[string]bool, err error) {
	orderedClients = make([]*clients.Client, 0)
	usedClientIDs = make(map[string]bool)

	for _, cid := range clientIDs {
		// TODO: doesn't GetByID exclude disconnected clients? if so, how can allowDisconnected work?
		client, err := al.clientService.GetByID(cid)
		if err != nil {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Failed to find a client with id=%q.", cid),
				Err:        err,
				HTTPStatus: http.StatusInternalServerError,
			}
			return orderedClients, usedClientIDs, err
		}
		if client == nil {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Client with id=%q not found.", cid),
				Err:        err,
				HTTPStatus: http.StatusNotFound,
			}
			return orderedClients, usedClientIDs, err
		}

		if client.DisconnectedAt != nil && !allowDisconnected {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Client with id=%q is not active.", cid),
				Err:        err,
				HTTPStatus: http.StatusBadRequest,
			}

			return orderedClients, usedClientIDs, err
		}

		usedClientIDs[cid] = true
		orderedClients = append(orderedClients, client)
	}
	return orderedClients, usedClientIDs, nil
}

func (al *APIListener) getOrderedClientsByTag(
	clientTags *models.JobClientTags,
	allowDisconnected bool,
) (
	orderedClients []*clients.Client,
	err error,
) {
	// find the clientIDs that have matching tags
	orderedClients, err = al.clientService.GetClientsByTag(clientTags.Tags, clientTags.Operator, allowDisconnected)
	if err != nil {
		err = errors2.APIError{
			Message:    "Unable to get active clients by tags",
			Err:        err,
			HTTPStatus: http.StatusInternalServerError,
		}
		return orderedClients, err
	}
	return orderedClients, err
}

func checkTargetingParams(params TargetingParams) (errTitle string, err error) {
	if params.GetClientIDs() == nil && params.GetGroupIDs() == nil && params.GetTags() == nil {
		return "Missing targeting parameters.", ErrRequestMissingTargetingParams
	}
	if params.GetClientIDs() != nil && params.GetTags() != nil ||
		params.GetGroupIDs() != nil && params.GetTags() != nil {
		return "Multiple targeting parameters.", ErrRequestIncludesMultipleTargetingParams
	}
	tags := params.GetTags()
	if tags != nil {
		if len(tags) == 0 {
			return "No tags specified.", ErrMissingTagsInMultiJobRequest
		}
	}
	return "", nil
}

func validateNonClientsTagTargeting(params TargetingParams, groupClientsCount int, orderedClients []*clients.Client, minClients int) (err error) {
	if len(params.GetGroupIDs()) > 0 && groupClientsCount == 0 && len(params.GetClientIDs()) == 0 {
		return errors.New("no active clients belong to the selected group(s)")
	}

	if len(params.GetClientIDs()) < minClients && groupClientsCount == 0 {
		return fmt.Errorf("at least %d clients should be specified", minClients)
	}

	if orderedClients != nil && len(orderedClients) == 0 {
		return fmt.Errorf("at least %d clients should be specified", minClients)
	}
	return nil
}

func validateClientTagsTargeting(orderedClients []*clients.Client) (err error) {
	minClients := 1
	if orderedClients == nil || len(orderedClients) < minClients {
		return fmt.Errorf(fmt.Sprintf("At least %d client should be specified.", minClients))
	}
	return nil
}

func hasClientTags(params TargetingParams) (has bool) {
	return params.GetTags() != nil
}

func (al *APIListener) makeJSONErr(w http.ResponseWriter, err error, errTitle string, isBadRequest bool) {
	if isBadRequest {
		if errTitle != "" {
			al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, "", errTitle, err.Error())
		} else {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, err.Error())
		}
	} else {
		al.jsonError(w, err)
	}
}
