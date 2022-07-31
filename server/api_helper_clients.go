package chserver

import (
	"context"
	"fmt"
	"net/http"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/models"
)

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
	ctx context.Context,
	clientIDs, groupIDs []string,
	clientTags *models.JobClientTags,
	allowDisconnected bool,
) (
	orderedClients []*clients.Client,
	err error,
) {
	// find the clientIDs that have matching tags
	orderedClients, err = al.clientService.GetActiveByTags(clientTags.Tags, clientTags.Operator)
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
