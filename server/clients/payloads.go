package clients

import (
	"time"

	"github.com/openrport/openrport/server/clients/clientdata"
	"github.com/openrport/openrport/server/clients/clienttunnel"
	"github.com/openrport/openrport/share/clientconfig"
	"github.com/openrport/openrport/share/models"
	"github.com/openrport/openrport/share/query"
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
	Inventory              **models.Inventory      `json:"inventory,omitempty"`
	IPAddresses            **models.IPAddresses    `json:"ext_ip_addresses,omitempty"`
	ClientConfiguration    **clientconfig.Config   `json:"client_configuration,omitempty"`
	Groups                 *[]string               `json:"groups,omitempty"`
	Labels                 *map[string]string      `json:"labels,omitempty"`
}

func ConvertToClientsPayload(clientsList []*clientdata.CalculatedClient, fields []query.FieldsOption) []ClientPayload {
	r := make([]ClientPayload, 0, len(clientsList))
	for _, cur := range clientsList {
		r = append(r, ConvertToClientPayload(cur, fields))
	}
	return r
}

func ConvertToClientPayload(client *clientdata.CalculatedClient, fields []query.FieldsOption) ClientPayload { //nolint:gocyclo
	requestedFields := query.RequestedFields(fields, "clients")
	p := ClientPayload{}
	for field := range OptionsSupportedFields["clients"] {
		if len(fields) > 0 && !requestedFields[field] {
			continue
		}

		client.GetLock().RLock()
		defer client.GetLock().RUnlock()

		switch field {
		case "id":
			id := client.ID
			p.ID = &id
		case "name":
			name := client.Name
			p.Name = &name
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
		case "labels":
			p.Labels = &client.Labels
		case "version":
			p.Version = &client.Version
		case "address":
			p.Address = &client.Address
		case "tunnels":
			p.Tunnels = &client.Tunnels
		case "disconnected_at":
			disconnectedAt := client.DisconnectedAt
			p.DisconnectedAt = &disconnectedAt
		case "last_heartbeat_at":
			lastHeartbeatAt := client.LastHeartbeatAt
			p.LastHeartbeatAt = &lastHeartbeatAt
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
		case "inventory":
			p.Inventory = &client.Inventory
		case "ip_addresses":
			p.IPAddresses = &client.IPAddresses
		case "client_configuration":
			p.ClientConfiguration = &client.ClientConfiguration
		case "groups":
			p.Groups = &client.Groups
		case "connection_state":
			connectionState := string(client.GetConnectionState())
			p.ConnectionState = &connectionState
		}
	}
	return p
}
