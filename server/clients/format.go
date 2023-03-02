package clients

import (
	"fmt"
)

func FormatConnectionState(client *Client) string {
	if !client.IsConnected() {
		return fmt.Sprintf("disconnected since %s", client.GetDisconnectedAtValue())
	}
	return "connected"
}
