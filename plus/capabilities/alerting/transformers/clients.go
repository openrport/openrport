package transformers

import (
	rportclients "github.com/realvnc-labs/rport/server/clients"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/clientupdates"
)

func TransformRportClientToClientUpdate(rc *rportclients.Client) (cl *clientupdates.Client, err error) {
	cl = &clientupdates.Client{}

	transformIdentity(rc, cl)
	transformConnectionInfo(rc, cl)
	transformMeta(rc, cl)
	transformUpdateStatus(rc, cl)
	transformClientDetails(rc, cl)

	return cl, nil
}

func transformIdentity(rc *rportclients.Client, cl *clientupdates.Client) {
	cl.Version = rc.GetVersion()
	cl.ID = rc.GetID()
	cl.Name = rc.GetName()
}

func transformConnectionInfo(rc *rportclients.Client, cl *clientupdates.Client) {
	cl.Address = rc.GetAddress()
	cl.ConnectionState = string(rc.CalculateConnectionState())
	disconnectedAt := rc.GetDisconnectedAtValue()
	cl.DisconnectedAt = &disconnectedAt
	lastHeartbeatAt := rc.GetLastHeartbeatAtValue()
	cl.LastHeartbeatAt = &lastHeartbeatAt
}

func transformMeta(rc *rportclients.Client, cl *clientupdates.Client) {
	cl.Tags = rc.GetTags()
	cl.Labels = rc.GetLabels()
	// cl.Groups = rc.Groups
}

func transformUpdateStatus(rc *rportclients.Client, cl *clientupdates.Client) {
	if rc.UpdatesStatus != nil {
		updatesStatus := rc.GetUpdatesStatus()
		cl.UpdatesAvailable = updatesStatus.UpdatesAvailable
		cl.SecurityUpdatesAvailable = updatesStatus.SecurityUpdatesAvailable
	}
}

func transformClientDetails(rc *rportclients.Client, cl *clientupdates.Client) {
	cl.Hostname = rc.GetHostname()
	cl.IPv4 = rc.GetIPv4()
	cl.IPv6 = rc.GetIPv6()
	cl.MemoryTotal = rc.GetMemoryTotal()
	cl.NumCPUs = rc.GetNumCPUs()
	cl.OS = rc.GetOS()
	cl.OSArch = rc.GetOSArch()
	cl.OSFamily = rc.GetOSFamily()
	cl.OSFullName = rc.GetOSFullName()
	cl.OSKernel = rc.GetOSKernel()
	cl.OSVersion = rc.GetOSVersion()
	cl.OSVirtualizationRole = rc.GetOSVirtualizationRole()
	cl.OSVirtualizationSystem = rc.GetOSVirtualizationSystem()
	cl.Timezone = rc.GetTimezone()
}
