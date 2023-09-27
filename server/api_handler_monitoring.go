package chserver

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/openrport/openrport/server/monitoring"
	"github.com/openrport/openrport/server/routes"
	"github.com/openrport/openrport/share/comm"
	"github.com/openrport/openrport/share/query"
)

// handleRefreshUpdatesStatus handles GET /clients/{client_id}/updates-status
func (al *APIListener) handleRefreshUpdatesStatus(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]
	if clientID == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "client id is missing")
		return
	}

	client, err := al.clientService.GetActiveByID(clientID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %s not found", clientID))
		return
	}

	err = comm.SendRequestAndGetResponse(client.GetConnection(), comm.RequestTypeRefreshUpdatesStatus, nil, nil, al.Log())
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetClientMetrics handles GET /clients/{client_id}/metrics
func (al *APIListener) handleGetClientMetrics(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]

	queryOptions := query.NewOptions(req, monitoring.ClientMetricsSortDefault, monitoring.ClientMetricsFilterDefault, monitoring.ClientMetricsFieldsDefault)

	payload, err := al.monitoringService.ListClientMetrics(req.Context(), clientID, queryOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("metrics for client with id %q not found", clientID))
			return
		}
		al.jsonError(w, err)
		return
	}
	al.writeJSONResponse(w, http.StatusOK, payload)
}

// handleGetClientGraphMetrics handles GET /clients/{client_id}/graph-metrics
func (al *APIListener) handleGetClientGraphMetrics(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]

	client, err := al.clientService.GetActiveByID(clientID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %s not found", clientID))
		return
	}

	queryOptions := query.NewOptions(req, monitoring.ClientGraphMetricsSortDefault, monitoring.ClientGraphMetricsFilterDefault, monitoring.ClientGraphMetricsFieldsDefault)
	requestInfo := query.ParseRequestInfo(req)

	monitoringConfig := client.GetMonitoringConfig()
	netLan := monitoringConfig.LanCard != nil
	netWan := monitoringConfig.WanCard != nil

	payload, err := al.monitoringService.ListClientGraphMetrics(req.Context(), clientID, queryOptions, requestInfo, netLan, netWan)
	if err != nil {
		if err == sql.ErrNoRows {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("graph-metrics for client with id %q not found", clientID))
			return
		}
		al.jsonError(w, err)
		return
	}
	al.writeJSONResponse(w, http.StatusOK, payload)
}

// handleGetClientGraphMetricsGraph handles /clients/{client_id}/graph-metrics/{"+routeParamGraphName+"}
func (al *APIListener) handleGetClientGraphMetricsGraph(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]
	graph := vars[routes.ParamGraphName]

	queryOptions := query.NewOptions(req, monitoring.ClientGraphMetricsSortDefault, monitoring.ClientGraphMetricsFilterDefault, monitoring.ClientGraphMetricsFieldsDefault)

	client, err := al.clientService.GetActiveByID(clientID)
	if err != nil {
		al.jsonErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %s not found", clientID))
		return
	}

	monitoringConfig := client.GetMonitoringConfig()

	payload, err := al.monitoringService.ListClientGraph(req.Context(), clientID, queryOptions, graph, monitoringConfig.LanCard, monitoringConfig.WanCard)
	if err != nil {
		if err == sql.ErrNoRows {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("graph-metrics for client with id %q not found", clientID))
			return
		}
		al.jsonError(w, err)
		return
	}
	al.writeJSONResponse(w, http.StatusOK, payload)
}

// handleGetClientProcesses handles GET /clients/{client_id}/processes
func (al *APIListener) handleGetClientProcesses(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]

	queryOptions := query.NewOptions(req, monitoring.ClientProcessesSortDefault, monitoring.ClientProcessesFilterDefault, monitoring.ClientProcessesFieldsDefault)

	payload, err := al.monitoringService.ListClientProcesses(req.Context(), clientID, queryOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("processes for client with id %q not found", clientID))
			return
		}
		al.jsonError(w, err)
		return
	}
	al.writeJSONResponse(w, http.StatusOK, payload)
}

// handleGetClientMountpoints handles GET /clients/{client_id}/mountpoints
func (al *APIListener) handleGetClientMountpoints(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]

	queryOptions := query.NewOptions(req, monitoring.ClientMountpointsSortDefault, monitoring.ClientMountpointsFilterDefault, monitoring.ClientMountpointsFieldsDefault)

	payload, err := al.monitoringService.ListClientMountpoints(req.Context(), clientID, queryOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("mountpoints for client with id %q not found", clientID))
			return
		}
		al.jsonError(w, err)
		return
	}
	al.writeJSONResponse(w, http.StatusOK, payload)
}

// handleMonitoringDisabled returns Not Found (404) when monitoring is disabled
func (al *APIListener) handleMonitoringDisabled(w http.ResponseWriter, req *http.Request) {
	al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "monitoring disabled. re-enable to view monitoring statistics.")
}
