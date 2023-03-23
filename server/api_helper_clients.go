package chserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	errors2 "github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/server/cgroups"
	"github.com/realvnc-labs/rport/server/clients"
	"github.com/realvnc-labs/rport/share/models"
)

const (
	minClientsForTargeting = 1
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
}

func (al *APIListener) getOrderedClientsWithValidation(
	ctx context.Context,
	params TargetingParams,
) (targetedClients []*clients.Client, groupClientsCount int, err error) {
	err = checkTargetingParams(params)
	if err != nil {
		return nil, 0, err
	}

	if !hasClientTags(params) {
		// do the original client ids flow
		targetedClients, groupClientsCount, err = al.getOrderedClients(ctx, params.GetClientIDs(), params.GetGroupIDs())
		if err != nil {
			return nil, 0, err
		}

		err := validateNonClientsTagTargeting(params, groupClientsCount, targetedClients)
		if err != nil {
			return nil, 0, err
		}
	} else {
		// do tags
		targetedClients, err = al.getOrderedClientsByTag(params.GetClientTags())
		if err != nil {
			return nil, 0, err
		}

		err := validateClientTagsTargeting(targetedClients)
		if err != nil {
			return nil, 0, err
		}
	}
	return targetedClients, groupClientsCount, nil
}

func (al *APIListener) getOrderedClients(
	ctx context.Context,
	clientIDs, groupIDs []string,
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

	orderedClients, usedClientIDs, err := al.makeClientsList(clientIDs)
	if err != nil {
		return orderedClients, groupClientsFoundCount, err
	}

	// append group clients
	for _, groupClient := range groupClients {
		groupClientID := groupClient.GetID()
		if !usedClientIDs[groupClientID] {
			usedClientIDs[groupClientID] = true
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
	return al.clientService.GetByGroups(groups)
}

func (al *APIListener) makeClientsList(clientIDs []string) (orderedClients []*clients.Client, usedClientIDs map[string]bool, err error) {
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

		usedClientIDs[cid] = true
		orderedClients = append(orderedClients, client)
	}
	return orderedClients, usedClientIDs, nil
}

func (al *APIListener) getOrderedClientsByTag(clientTags *models.JobClientTags) (
	orderedClients []*clients.Client,
	err error,
) {
	// find the clientIDs that have matching tags
	orderedClients, err = al.clientService.GetClientsByTag(clientTags.Tags, clientTags.Operator, true /* allowDisconnected */)
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

func checkTargetingParams(params TargetingParams) (err error) {
	if params.GetClientIDs() == nil && params.GetGroupIDs() == nil && params.GetClientTags() == nil {
		return errors2.APIError{
			Message:    "Missing targeting parameters.",
			Err:        ErrRequestMissingTargetingParams,
			HTTPStatus: http.StatusBadRequest,
		}
	}
	if params.GetClientIDs() != nil && params.GetClientTags() != nil ||
		params.GetGroupIDs() != nil && params.GetClientTags() != nil {
		return errors2.APIError{
			Message:    "Multiple targeting parameters.",
			Err:        ErrRequestIncludesMultipleTargetingParams,
			HTTPStatus: http.StatusBadRequest,
		}
	}
	clientTags := params.GetClientTags()
	if clientTags != nil {
		if len(clientTags.Tags) == 0 {
			return errors2.APIError{
				Message:    "No tags specified.",
				Err:        ErrMissingTagsInMultiJobRequest,
				HTTPStatus: http.StatusBadRequest,
			}
		}
	}
	return nil
}

func validateNonClientsTagTargeting(params TargetingParams, groupClientsCount int, orderedClients []*clients.Client) (err error) {
	if len(params.GetGroupIDs()) > 0 && groupClientsCount == 0 && len(params.GetClientIDs()) == 0 {
		return errors2.APIError{
			Err:        errors.New("no active clients belong to the selected group(s)"),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	if len(params.GetClientIDs()) < minClientsForTargeting && groupClientsCount == 0 {
		return errors2.APIError{
			Err:        errors.New("at least 1 client should be specified"),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	if len(orderedClients) == 0 {
		return errors2.APIError{
			Err:        errors.New("at least 1 client should be specified"),
			HTTPStatus: http.StatusBadRequest,
		}
	}
	return nil
}

func validateClientTagsTargeting(orderedClients []*clients.Client) (err error) {
	if orderedClients == nil || len(orderedClients) < minClientsForTargeting {
		return errors2.APIError{
			Err:        errors.New("at least 1 client should be specified"),
			HTTPStatus: http.StatusBadRequest,
		}
	}
	return nil
}

func hasClientTags(params TargetingParams) (has bool) {
	return params.GetClientTags() != nil
}
