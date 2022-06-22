package chserver

import (
	"context"
	"fmt"
	"net/http"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
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
	var groups []*cgroups.ClientGroup
	for _, groupID := range groupIDs {
		group, err := al.clientGroupProvider.Get(ctx, groupID)
		if err != nil {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Failed to get a client group with id=%q.", groupID),
				Err:        err,
				HTTPStatus: http.StatusInternalServerError,
			}
			return orderedClients, groupClientsFoundCount, err
		}
		if group == nil {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Unknown group with id=%q.", groupID),
				Err:        err,
				HTTPStatus: http.StatusBadRequest,
			}
			return orderedClients, 0, err
		}
		groups = append(groups, group)
	}
	groupClients := al.clientService.GetActiveByGroups(groups)
	groupClientsFoundCount = len(groupClients)

	orderedClients = make([]*clients.Client, 0)
	usedClientIDs := make(map[string]bool)
	for _, cid := range clientIDs {
		client, err := al.clientService.GetByID(cid)
		if err != nil {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Failed to find a client with id=%q.", cid),
				Err:        err,
				HTTPStatus: http.StatusInternalServerError,
			}
			return orderedClients, 0, err
		}
		if client == nil {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Client with id=%q not found.", cid),
				Err:        err,
				HTTPStatus: http.StatusNotFound,
			}
			return orderedClients, 0, err
		}

		if client.DisconnectedAt != nil && !allowDisconnected {
			err = errors2.APIError{
				Message:    fmt.Sprintf("Client with id=%q is not active.", cid),
				Err:        err,
				HTTPStatus: http.StatusBadRequest,
			}

			return orderedClients, 0, err
		}

		usedClientIDs[cid] = true
		orderedClients = append(orderedClients, client)
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
