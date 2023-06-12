package chserver

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/ssh"

	"github.com/realvnc-labs/rport/server/api"
	apierrors "github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/server/auditlog"
	"github.com/realvnc-labs/rport/server/clients"
	"github.com/realvnc-labs/rport/server/clients/clienttunnel"
	"github.com/realvnc-labs/rport/server/ports"
	"github.com/realvnc-labs/rport/server/routes"
	"github.com/realvnc-labs/rport/server/validation"
	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/models"
	"github.com/realvnc-labs/rport/share/query"
)

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

	clientPayload := clients.ConvertToClientPayload(client.ToCalculated(groups), options.Fields)
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

	filteredClients, err := al.clientService.GetFilteredUserClients(curUser, options.Filters, groups)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	sortFunc(filteredClients, desc)

	totalCount := len(filteredClients)
	start, end := options.Pagination.GetStartEnd(totalCount)
	filteredClients = filteredClients[start:end]

	clientsPayload := clients.ConvertToClientsPayload(filteredClients, options.Fields)

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

	if client.IsPaused() {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, fmt.Sprintf("failed to start tunnel for client with id %s due to client being paused (reason = %s)", clientID, client.GetPausedReason()))
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

	client.Log().Debugf("requested remote = %#v", remote)

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

	allowed, err := clienttunnel.IsAllowed(remote.Remote(), client.GetConnection(), al.Log())
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

	for _, t := range client.GetTunnels() {
		if t.Remote.Remote() == remote.Remote() && t.Remote.IsProtocol(remote.Protocol) && t.EqualACL(remote.ACL) {
			al.jsonErrorResponseWithErrCode(w, http.StatusBadRequest, ErrCodeTunnelToPortExist, fmt.Sprintf("Tunnel to port %s already exists.", remote.RemotePort))
			return
		}
	}

	if checkPortStr := req.URL.Query().Get("check_port"); checkPortStr != "0" && remote.IsProtocol(models.ProtocolTCP) {
		err = al.checkRemotePort(*remote, client.GetConnection())
		if err != nil {
			al.jsonError(w, err)
			return
		}
	}

	if remote.IsLocalSpecified() {
		err = al.checkLocalPort(remote.LocalPort, remote.Protocol)
		if err != nil {
			al.jsonError(w, err)
			return
		}
	}

	// populating tunnel (remote) ownership
	currUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}
	remote.Owner = currUser.Username

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

	if isHTTPProxy && !al.config.Server.InternalTunnelProxyConfig.Enabled {
		return apierrors.NewAPIError(http.StatusBadRequest, "", "creation of tunnel proxy not enabled", nil)
	}
	if isHTTPProxy && remote.Scheme != nil && !validation.SchemeSupportsHTTPProxy(*remote.Scheme) {
		return apierrors.NewAPIError(http.StatusBadRequest, "", fmt.Sprintf("tunnel proxy not allowed with scheme %s", *remote.Scheme), nil)
	}
	if isHTTPProxy && !remote.IsProtocol(models.ProtocolTCP) {
		return apierrors.NewAPIError(http.StatusBadRequest, "", fmt.Sprintf("tunnel proxy not allowed with protcol %s", remote.Protocol), nil)
	}

	if isHTTPProxy && al.config.CaddyEnabled() {
		downstreamSubdomain, err := al.config.Caddy.SubDomainGenerator.GetRandomSubdomain()
		if err != nil {
			return apierrors.NewAPIError(http.StatusInternalServerError, "", "failed to allocate random subdomain for downstream proxy", err)
		}

		_, port, err := net.SplitHostPort(al.config.Caddy.HostAddress)
		if err != nil {
			return apierrors.NewAPIError(http.StatusInternalServerError, "", "failed to get host port from caddy config", err)
		}

		remote.TunnelURL = remote.NewDownstreamProxyURL(downstreamSubdomain, al.config.Caddy.BaseDomain, port)
	}

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

// TODO: remove this check, do it in client srv in startClientTunnels when https://github.com/realvnc-labs/rport/pull/252 will be in master.
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
	err = comm.SendRequestAndGetResponse(conn, comm.RequestTypeCheckPort, req, resp, al.Log())
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

	tunnel := al.clientService.FindTunnel(client, tunnelID)
	if tunnel == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "tunnel not found")
		return
	}

	curUser, err := al.getUserModelForAuth(req.Context())
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.extendedPermissionDeleteTunnelRaw(tunnel, curUser)
	if err != nil {
		al.jsonError(w, err)
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

func (al *APIListener) handlePutClientTunnelACL(w http.ResponseWriter, req *http.Request) {
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

	tunnelID := vars["tunnel_id"]
	if tunnelID == "" {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "tunnel id is missing")
		return
	}

	tunnel := al.clientService.FindTunnel(client, tunnelID)
	if tunnel == nil {
		al.jsonErrorResponseWithTitle(w, http.StatusNotFound, "tunnel not found")
		return
	}

	var reqBody struct {
		ACL *string `json:"acl"`
	}
	err = parseRequestBody(req.Body, &reqBody)
	if err != nil {
		al.jsonError(w, err)
		return
	}

	err = al.clientService.SetTunnelACL(client, tunnel, reqBody.ACL)
	if err != nil {
		al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, err.Error())
		return
	}

	al.auditLog.Entry(auditlog.ApplicationClientTunnel, auditlog.ActionUpdate).
		WithHTTPRequest(req).
		WithClient(client).
		WithID(tunnelID).
		WithRequest(reqBody).
		Save()

	w.WriteHeader(http.StatusNoContent)
}
