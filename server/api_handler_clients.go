package chserver

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/api"
	apierrors "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/clients/clienttunnel"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/routes"
	"github.com/cloudradar-monitoring/rport/server/validation"
	"github.com/cloudradar-monitoring/rport/share/clientconfig"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type ClientPayload struct {
	ID                     *string                 `json:"id,omitempty"`
	Name                   *string                 `json:"name,omitempty"`
	Address                *string                 `json:"address,omitempty"`
	Hostname               *string                 `json:"hostname,omitempty"`
	OS                     *string                 `json:"os,omitempty"`
	OSFullName             *string                 `json:"os_full_name,omitempty"`
	OSVersion              *string                 `json:"os_version,omitempty"`
	OSArch                 *string                 `json:"os_arch,omitempty"`
	OSFamily               *string                 `json:"os_family,omitempty"`
	OSKernel               *string                 `json:"os_kernel,omitempty"`
	OSVirtualizationSystem *string                 `json:"os_virtualization_system,omitempty"`
	OSVirtualizationRole   *string                 `json:"os_virtualization_role,omitempty"`
	NumCPUs                *int                    `json:"num_cpus,omitempty"`
	CPUFamily              *string                 `json:"cpu_family,omitempty"`
	CPUModel               *string                 `json:"cpu_model,omitempty"`
	CPUModelName           *string                 `json:"cpu_model_name,omitempty"`
	CPUVendor              *string                 `json:"cpu_vendor,omitempty"`
	MemoryTotal            *uint64                 `json:"mem_total,omitempty"`
	Timezone               *string                 `json:"timezone,omitempty"`
	ClientAuthID           *string                 `json:"client_auth_id,omitempty"`
	Version                *string                 `json:"version,omitempty"`
	DisconnectedAt         **time.Time             `json:"disconnected_at,omitempty"`
	LastHeartbeatAt        **time.Time             `json:"last_heartbeat_at,omitempty"`
	ConnectionState        *string                 `json:"connection_state,omitempty"`
	IPv4                   *[]string               `json:"ipv4,omitempty"`
	IPv6                   *[]string               `json:"ipv6,omitempty"`
	Tags                   *[]string               `json:"tags,omitempty"`
	AllowedUserGroups      *[]string               `json:"allowed_user_groups,omitempty"`
	Tunnels                *[]*clienttunnel.Tunnel `json:"tunnels,omitempty"`
	UpdatesStatus          **models.UpdatesStatus  `json:"updates_status,omitempty"`
	ClientConfiguration    **clientconfig.Config   `json:"client_configuration,omitempty"`
	Groups                 *[]string               `json:"groups,omitempty"`
}

func convertToClientsPayload(clients []*clients.CalculatedClient, fields []query.FieldsOption) []ClientPayload {
	r := make([]ClientPayload, 0, len(clients))
	for _, cur := range clients {
		r = append(r, convertToClientPayload(cur, fields))
	}
	return r
}

func convertToClientPayload(client *clients.CalculatedClient, fields []query.FieldsOption) ClientPayload { //nolint:gocyclo
	requestedFields := query.RequestedFields(fields, "clients")
	p := ClientPayload{}
	for field := range clients.OptionsSupportedFields["clients"] {
		if len(fields) > 0 && !requestedFields[field] {
			continue
		}
		switch field {
		case "id":
			p.ID = &client.ID
		case "name":
			p.Name = &client.Name
		case "os":
			p.OS = &client.OS
		case "os_arch":
			p.OSArch = &client.OSArch
		case "os_family":
			p.OSFamily = &client.OSFamily
		case "os_kernel":
			p.OSKernel = &client.OSKernel
		case "hostname":
			p.Hostname = &client.Hostname
		case "ipv4":
			p.IPv4 = &client.IPv4
		case "ipv6":
			p.IPv6 = &client.IPv6
		case "tags":
			p.Tags = &client.Tags
		case "version":
			p.Version = &client.Version
		case "address":
			p.Address = &client.Address
		case "tunnels":
			p.Tunnels = &client.Tunnels
		case "disconnected_at":
			p.DisconnectedAt = &client.DisconnectedAt
		case "last_heartbeat_at":
			p.LastHeartbeatAt = &client.LastHeartbeatAt
		case "connection_state":
			connectionState := string(client.ConnectionState)
			p.ConnectionState = &connectionState
		case "client_auth_id":
			p.ClientAuthID = &client.ClientAuthID
		case "os_full_name":
			p.OSFullName = &client.OSFullName
		case "os_version":
			p.OSVersion = &client.OSVersion
		case "os_virtualization_system":
			p.OSVirtualizationSystem = &client.OSVirtualizationSystem
		case "os_virtualization_role":
			p.OSVirtualizationRole = &client.OSVirtualizationRole
		case "cpu_family":
			p.CPUFamily = &client.CPUFamily
		case "cpu_model":
			p.CPUModel = &client.CPUModel
		case "cpu_model_name":
			p.CPUModelName = &client.CPUModelName
		case "cpu_vendor":
			p.CPUVendor = &client.CPUVendor
		case "timezone":
			p.Timezone = &client.Timezone
		case "num_cpus":
			p.NumCPUs = &client.NumCPUs
		case "mem_total":
			p.MemoryTotal = &client.MemoryTotal
		case "allowed_user_groups":
			p.AllowedUserGroups = &client.AllowedUserGroups
		case "updates_status":
			p.UpdatesStatus = &client.UpdatesStatus
		case "client_configuration":
			p.ClientConfiguration = &client.ClientConfiguration
		case "groups":
			p.Groups = &client.Groups
		}
	}
	return p
}

func getCorrespondingSortFunc(sorts []query.SortOption) (sortFunc func(a []*clients.CalculatedClient, desc bool), desc bool, err error) {
	if len(sorts) < 1 {
		return clients.SortByID, false, nil
	}
	if len(sorts) > 1 {
		return nil, false, apierrors.APIError{
			Message:    "Only one sort field is supported for clients.",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	switch sorts[0].Column {
	case "id":
		sortFunc = clients.SortByID
	case "name":
		sortFunc = clients.SortByName
	case "os":
		sortFunc = clients.SortByOS
	case "hostname":
		sortFunc = clients.SortByHostname
	case "version":
		sortFunc = clients.SortByVersion
	}

	return sortFunc, !sorts[0].IsASC, nil
}

func (al *APIListener) handleGetClient(w http.ResponseWriter, req *http.Request) {
	options := query.GetRetrieveOptions(req)
	errs := query.ValidateRetrieveOptions(options, clients.OptionsSupportedFields)
	if errs != nil {
		al.jsonError(w, errs)
		return
	}

	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]

	client, err := al.clientService.GetByID(clientID)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if client == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("client with id %q not found", clientID))
		return
	}

	groups, err := al.clientGroupProvider.GetAll(req.Context())
	if err != nil {
		al.jsonErrorResponseWithError(w, http.StatusInternalServerError, "Failed to get client groups.", err)
		return
	}

	clientPayload := convertToClientPayload(client.ToCalculated(groups), options.Fields)
	al.writeJSONResponse(w, http.StatusOK, api.NewSuccessPayload(clientPayload))
}

func (al *APIListener) handleDeleteClient(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]
	err := al.clientService.DeleteOffline(clientID)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationClient, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithID(clientID).
		Save()

	w.WriteHeader(http.StatusNoContent)
	al.Debugf("Client %q deleted.", clientID)
}

type clientACLRequest struct {
	AllowedUserGroups []string `json:"allowed_user_groups"`
}

func (al *APIListener) handlePostClientACL(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cid := vars[routes.ParamClientID]
	if cid == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routes.ParamClientID))
		return
	}

	var reqBody clientACLRequest
	err := parseRequestBody(req.Body, &reqBody)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.userService.ExistGroups(reqBody.AllowedUserGroups)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.clientService.SetACL(cid, reqBody.AllowedUserGroups)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	al.auditLog.Entry(auditlog.ApplicationClientACL, auditlog.ActionUpdate).
		WithHTTPRequest(req).
		WithID(cid).
		WithRequest(reqBody).
		Save()

	w.WriteHeader(http.StatusNoContent)
}

func (al *APIListener) handleGetClients(w http.ResponseWriter, req *http.Request) {
	options := query.NewOptions(req, nil, nil, clients.OptionsListDefaultFields)
	errs := query.ValidateListOptions(options, clients.OptionsSupportedSorts, clients.OptionsSupportedFilters, clients.OptionsSupportedFields, &query.PaginationConfig{
		MaxLimit:     500,
		DefaultLimit: 50,
	})
	if errs != nil {
		al.jsonError(w, errs)
		return
	}

	sortFunc, desc, err := getCorrespondingSortFunc(options.Sorts)
	if err != nil {
		al.jsonError(w, err)
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

	cls, err := al.clientService.GetFilteredUserClients(curUser, options.Filters, groups)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	sortFunc(cls, desc)

	totalCount := len(cls)
	start, end := options.Pagination.GetStartEnd(totalCount)
	cls = cls[start:end]

	clientsPayload := convertToClientsPayload(cls, options.Fields)
	al.writeJSONResponse(w, http.StatusOK, &api.SuccessPayload{
		Data: clientsPayload,
		Meta: api.NewMeta(totalCount),
	})
}

const (
	URISchemeMaxLength = 15

	autoCloseQueryParam          = "auto-close"
	idleTimeoutMinutesQueryParam = "idle-timeout-minutes"
	skipIdleTimeoutQueryParam    = "skip-idle-timeout"

	ErrCodeLocalPortInUse        = "ERR_CODE_LOCAL_PORT_IN_USE"
	ErrCodeRemotePortNotOpen     = "ERR_CODE_REMOTE_PORT_NOT_OPEN"
	ErrCodeTunnelExist           = "ERR_CODE_TUNNEL_EXIST"
	ErrCodeTunnelToPortExist     = "ERR_CODE_TUNNEL_TO_PORT_EXIST"
	ErrCodeURISchemeLengthExceed = "ERR_CODE_URI_SCHEME_LENGTH_EXCEED"
	ErrCodeInvalidACL            = "ERR_CODE_INVALID_ACL"
)

func (al *APIListener) handlePutClientTunnel(w http.ResponseWriter, req *http.Request) {
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

	localAddr := req.URL.Query().Get("local")
	remoteAddr := req.URL.Query().Get("remote")

	remoteStr := localAddr + ":" + remoteAddr
	if localAddr == "" {
		remoteStr = remoteAddr
	}

	protocol := req.URL.Query().Get("protocol")
	if protocol != "" {
		remoteStr += "/" + protocol
	}

	remote, err := models.NewRemote(remoteStr)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("failed to decode %q: %v", remoteStr, err))
		return
	}

	name := req.URL.Query().Get("name")
	if name != "" {
		remote.Name = name
	}

	schemeStr := req.URL.Query().Get("scheme")
	if len(schemeStr) > URISchemeMaxLength {
		al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, ErrCodeURISchemeLengthExceed, "Invalid URI scheme.", "Exceeds the max length.")
		return
	}
	if schemeStr != "" {
		remote.Scheme = &schemeStr
	}

	err = al.setTunnelProxyOptionsForRemote(req, remote)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.setAuthOptionsForRemote(req, remote)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.setAutoCloseIdleOptionsForRemote(req, remote)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	aclStr := req.URL.Query().Get("acl")
	if _, err = clienttunnel.ParseTunnelACL(aclStr); err != nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeInvalidACL, fmt.Sprintf("Invalid ACL: %s", err))
		return
	}
	if aclStr != "" {
		remote.ACL = &aclStr
	}

	allowed, err := clienttunnel.IsAllowed(remote.Remote(), client.Connection)
	if err != nil {
		al.jsonError(w, err)
		return
	}
	if !allowed {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "Tunnel destination is not allowed by client configuration.")
		return
	}

	if existing := al.clientService.FindTunnelByRemote(client, remote); existing != nil {
		al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeTunnelExist, "Tunnel already exist.")
		return
	}

	for _, t := range client.Tunnels {
		if t.Remote.Remote() == remote.Remote() && t.Remote.IsProtocol(remote.Protocol) && t.EqualACL(remote.ACL) {
			al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeTunnelToPortExist, fmt.Sprintf("Tunnel to port %s already exists.", remote.RemotePort))
			return
		}
	}

	if checkPortStr := req.URL.Query().Get("check_port"); checkPortStr != "0" && remote.IsProtocol(models.ProtocolTCP) {
		err = al.checkRemotePort(*remote, client.Connection)
		if err != nil {
			al.jsonError(w, err)
			return
		}
	}

	// make next steps thread-safe
	client.Lock()
	defer client.Unlock()

	if remote.IsLocalSpecified() {
		err = al.checkLocalPort(remote.LocalPort, remote.Protocol)
		if err != nil {
			al.jsonError(w, err)
			return
		}
	}

	// start the new tunnel only
	tunnels, err := al.clientService.StartClientTunnels(client, []*models.Remote{remote})
	if err != nil {
		al.jsonError(w, err)
		return
	}
	response := api.NewSuccessPayload(tunnels[0])

	al.auditLog.Entry(auditlog.ApplicationClientTunnel, auditlog.ActionCreate).
		WithHTTPRequest(req).
		WithClient(client).
		WithRequest(remote).
		WithResponse(tunnels[0]).
		WithID(tunnels[0].ID).
		Save()

	al.writeJSONResponse(w, http.StatusOK, response)
}

func (al *APIListener) setTunnelProxyOptionsForRemote(req *http.Request, remote *models.Remote) (err error) {
	httpProxy := req.URL.Query().Get("http_proxy")
	if httpProxy == "" {
		httpProxy = "false"
	}
	isHTTPProxy, err := strconv.ParseBool(httpProxy)
	if err != nil {
		return err
	}

	shouldUseSubdomain := req.URL.Query().Get("use_subdomain")
	if shouldUseSubdomain == "" {
		shouldUseSubdomain = "false"
	}
	useSubdomain, err := strconv.ParseBool(shouldUseSubdomain)
	if err != nil {
		return err
	}

	if isHTTPProxy && !al.config.Server.InternalTunnelProxyConfig.Enabled {
		return apierrors.NewAPIError(http.StatusBadRequest, "", "creation of tunnel proxy not enabled", nil)
	}
	if isHTTPProxy && !validation.SchemeSupportsHTTPProxy(*remote.Scheme) {
		return apierrors.NewAPIError(http.StatusBadRequest, "", fmt.Sprintf("tunnel proxy not allowed with scheme %s", *remote.Scheme), nil)
	}
	if isHTTPProxy && !remote.IsProtocol(models.ProtocolTCP) {
		return apierrors.NewAPIError(http.StatusBadRequest, "", fmt.Sprintf("tunnel proxy not allowed with protcol %s", remote.Protocol), nil)
	}

	// TODO: (rs): add tests for when use_subdomain specified
	if useSubdomain {
		if !al.config.CaddyConfigured() {
			return apierrors.NewAPIError(http.StatusBadRequest, "", "when using use_subdomain, subdomain tunnels must be configured", nil)
		}
		if !isHTTPProxy {
			return apierrors.NewAPIError(http.StatusBadRequest, "", "when using use_subdomain, http_proxy must be specified", nil)
		}
	}

	remote.UseLocalSubdomain = useSubdomain
	remote.HTTPProxy = isHTTPProxy

	hostHeader := req.URL.Query().Get("host_header")
	if hostHeader != "" {
		if isHTTPProxy {
			remote.HostHeader = hostHeader
		} else {
			return apierrors.NewAPIError(http.StatusBadRequest, "", "host_header not allowed when http_proxy is false", nil)
		}
	}

	return err
}

func (al *APIListener) setAuthOptionsForRemote(req *http.Request, remote *models.Remote) (err error) {
	authUser := req.URL.Query().Get("auth_user")
	authPassword := req.URL.Query().Get("auth_password")
	if authUser != "" || authPassword != "" {
		if !remote.HTTPProxy {
			return apierrors.NewAPIError(http.StatusBadRequest, "", "http basic authentication requires http_proxy to be activated on the requested tunnel", nil)
		}
		if authPassword != "" && authUser == "" {
			return apierrors.NewAPIError(http.StatusBadRequest, "", "auth_password requires auth_user", nil)
		}
		if authUser != "" && authPassword == "" {
			return apierrors.NewAPIError(http.StatusBadRequest, "", "auth_user requires auth_password", nil)
		}
		remote.AuthUser = authUser
		remote.AuthPassword = authPassword
	}

	return err
}

func (al *APIListener) setAutoCloseIdleOptionsForRemote(req *http.Request, remote *models.Remote) (err error) {
	idleTimeoutMinutesStr := req.URL.Query().Get(idleTimeoutMinutesQueryParam)
	skipIdleTimeout, err := strconv.ParseBool(req.URL.Query().Get(skipIdleTimeoutQueryParam))
	if err != nil {
		skipIdleTimeout = false
	}

	idleTimeout, err := validation.ResolveIdleTunnelTimeoutValue(idleTimeoutMinutesStr, skipIdleTimeout)
	if err != nil {
		return err
	}

	remote.IdleTimeoutMinutes = int(idleTimeout.Minutes())

	remote.AutoClose, err = validation.ResolveTunnelAutoCloseValue(req.URL.Query().Get(autoCloseQueryParam))
	if err != nil {
		return err
	}

	return err
}

// TODO: remove this check, do it in client srv in startClientTunnels when https://github.com/cloudradar-monitoring/rport/pull/252 will be in master.
// APIError needs both httpStatusCode and errorCode. To avoid too many merge conflicts with PR252 temporarily use this check to avoid breaking UI
func (al *APIListener) checkLocalPort(localPort, protocol string) (err error) {
	lport, err := strconv.Atoi(localPort)
	if err != nil {
		return apierrors.NewAPIError(http.StatusBadRequest, "", fmt.Sprintf("Invalid port: %s.", localPort), err)
	}

	busyPorts, err := ports.ListBusyPorts(protocol)
	if err != nil {
		return apierrors.NewAPIError(http.StatusInternalServerError, "", "", err)
	}

	if busyPorts.Contains(lport) {
		return apierrors.NewAPIError(http.StatusBadRequest, ErrCodeLocalPortInUse, fmt.Sprintf("Port %d already in use.", lport), nil)
	}

	return nil
}

func (al *APIListener) checkRemotePort(remote models.Remote, conn ssh.Conn) (err error) {
	req := &comm.CheckPortRequest{
		HostPort: remote.Remote(),
		Timeout:  al.config.Server.CheckPortTimeout,
	}
	resp := &comm.CheckPortResponse{}
	err = comm.SendRequestAndGetResponse(conn, comm.RequestTypeCheckPort, req, resp)
	if err != nil {
		if _, ok := err.(*comm.ClientError); ok {
			err = apierrors.NewAPIError(http.StatusConflict, "", "", err)
		} else {
			err = apierrors.NewAPIError(http.StatusInternalServerError, "", "", err)
		}
		return err
	}

	if !resp.Open {
		err := apierrors.NewAPIError(
			http.StatusBadRequest,
			ErrCodeRemotePortNotOpen,
			fmt.Sprintf("Port %s is not in listening state.", remote.RemotePort),
			errors.New(resp.ErrMsg),
		)
		return err
	}

	return nil
}

func (al *APIListener) handleDeleteClientTunnel(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	clientID := vars[routes.ParamClientID]
	if clientID == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "client id is missing")
		return
	}

	force := false
	forceStr := req.URL.Query().Get("force")
	if forceStr != "" {
		var err error
		force, err = strconv.ParseBool(forceStr)
		if err != nil {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Invalid force param: %v.", forceStr))
			return
		}
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

	tunnelID := vars["tunnel_id"]
	if tunnelID == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "tunnel id is missing")
		return
	}

	// make next steps thread-safe
	client.Lock()
	defer client.Unlock()

	tunnel := al.clientService.FindTunnel(client, tunnelID)
	if tunnel == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "tunnel not found")
		return
	}

	err = al.clientService.TerminateTunnel(client, tunnel, force)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusConflict, err.Error())
		return
	}

	al.auditLog.Entry(auditlog.ApplicationClientTunnel, auditlog.ActionDelete).
		WithHTTPRequest(req).
		WithClient(client).
		WithID(tunnelID).
		WithRequest(map[string]interface{}{
			"force": force,
		}).
		Save()

	w.WriteHeader(http.StatusNoContent)
}
