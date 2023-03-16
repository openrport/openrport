package chserver

import (
	"net/http"
	"sort"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/share/query"
)

var (
	ClientTagsOptionsSupportedFields = map[string]map[string]bool{
		"client_tags": {
			"tag":        true,
			"client_ids": true,
		},
	}
)

type ClientTagPayload struct {
	Tag       *string   `json:"tag,omitempty"`
	ClientIDs *[]string `json:"client_ids,omitempty"`

	tagForSort string `json:"-"`
}

func (al *APIListener) handleGetClientTags(w http.ResponseWriter, req *http.Request) {
	options := query.GetListOptions(req)
	errs := query.ValidateListOptions(options, nil /* sorts */, nil /* filters */, ClientTagsOptionsSupportedFields, &query.PaginationConfig{
		MaxLimit:     500,
		DefaultLimit: 50,
	})
	if errs != nil {
		al.jsonError(w, errs)
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	groups, err := al.clientGroupProvider.GetAll(req.Context())
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to get client groups.", err)
		return
	}

	clients := al.clientService.GetUserClients(groups, curUser)

	clientIDsByTag := make(map[string][]string)
	for _, client := range clients {
		for _, tag := range client.Tags {
			clientIDsByTag[tag] = append(clientIDsByTag[tag], client.ID)
		}
	}

	payload := convertToClientTagsPayload(clientIDsByTag, options.Fields)
	sort.Slice(payload, func(i, j int) bool { return payload[i].tagForSort < payload[j].tagForSort })

	totalCount := len(payload)
	start, end := options.Pagination.GetStartEnd(totalCount)
	payload = payload[start:end]

	al.writeJSONResponse(w, http.StatusOK, &api.SuccessPayload{
		Data: payload,
		Meta: api.NewMeta(totalCount),
	})
}

func convertToClientTagsPayload(data map[string][]string, fields []query.FieldsOption) []ClientTagPayload {
	result := make([]ClientTagPayload, 0, len(data))
	for tag, clientIDs := range data {
		result = append(result, convertToClientTagPayload(tag, clientIDs, fields))
	}
	return result
}

func convertToClientTagPayload(tag string, clientIDs []string, fields []query.FieldsOption) ClientTagPayload { //nolint:gocyclo
	requestedFields := query.RequestedFields(fields, "client_tags")
	payload := ClientTagPayload{
		tagForSort: tag,
	}
	for field := range ClientTagsOptionsSupportedFields["client_tags"] {
		if len(fields) > 0 && !requestedFields[field] {
			continue
		}

		switch field {
		case "tag":
			payload.Tag = &tag
		case "client_ids":
			payload.ClientIDs = &clientIDs
		}
	}
	return payload
}
