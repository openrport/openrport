package clients

import (
	"fmt"

	"github.com/openrport/openrport/server/clients/clientdata"
)

func FormatConnectionState(client *clientdata.Client) string {
	if !client.IsConnected() {
		return fmt.Sprintf("disconnected since %s", client.GetDisconnectedAtValue())
	}
	return "connected"
}
