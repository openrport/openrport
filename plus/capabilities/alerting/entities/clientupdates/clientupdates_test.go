package clientupdates

import (
	"reflect"
	"testing"
	"time"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/measures"
)

func TestCloneClient(t *testing.T) {
	// Create a sample Client instance
	client := &Client{
		Timestamp:                time.Now(),
		UID:                      "123",
		Version:                  "1.0",
		ID:                       "456",
		Name:                     "John",
		Address:                  "123 Main St",
		DisconnectedAt:           nil,
		LastHeartbeatAt:          nil,
		ConnectionState:          "connected",
		Tags:                     []string{"tag1", "tag2"},
		Labels:                   map[string]string{"label1": "value1", "label2": "value2"},
		UpdatesAvailable:         1,
		SecurityUpdatesAvailable: 0,
		OS:                       "Linux",
		OSArch:                   "amd64",
		OSFamily:                 "debian",
		OSKernel:                 "4.19.0-10-amd64",
		OSFullName:               "Debian GNU/Linux 10 (buster)",
		OSVersion:                "10",
		OSVirtualizationSystem:   "docker",
		OSVirtualizationRole:     "host",
		NumCPUs:                  4,
		MemoryTotal:              8192,
		Timezone:                 "UTC",
		Hostname:                 "localhost",
		IPv4:                     []string{"192.168.0.1", "192.168.0.2"},
		IPv6:                     []string{"2001:db8::1", "2001:db8::2"},
		Measurements:             measures.Measures{},
	}

	clonedClient := client.Clone()

	if &clonedClient == client {
		t.Errorf("Clone() did not create a new instance")
	}

	if !reflect.DeepEqual(clonedClient, *client) {
		t.Errorf("Cloned client is not equal to the original client")
	}

	if &clonedClient.Tags == &client.Tags || !reflect.DeepEqual(clonedClient.Tags, client.Tags) {
		t.Errorf("Tags field of cloned client is not a new instance or has different elements")
	}

	if &clonedClient.Labels == &client.Labels || !reflect.DeepEqual(clonedClient.Labels, client.Labels) {
		t.Errorf("Labels field of cloned client is not a new instance or has different elements")
	}

	if &clonedClient.IPv4 == &client.IPv4 || !reflect.DeepEqual(clonedClient.IPv4, client.IPv4) {
		t.Errorf("IPv4 field of cloned client is not a new instance or has different elements")
	}

	if &clonedClient.IPv6 == &client.IPv6 || !reflect.DeepEqual(clonedClient.IPv6, client.IPv6) {
		t.Errorf("IPv6 field of cloned client is not a new instance or has different elements")
	}

	if &clonedClient.Measurements == &client.Measurements || !reflect.DeepEqual(clonedClient.Measurements, client.Measurements) {
		t.Errorf("Measures field of cloned client is not a new instance or has different values")
	}

	if client.DisconnectedAt != nil && clonedClient.DisconnectedAt == nil {
		t.Errorf("DisconnectedAt field of cloned client should not be nil")
	}

	if client.LastHeartbeatAt != nil && clonedClient.LastHeartbeatAt == nil {
		t.Errorf("LastHeartbeatAt field of cloned client should not be nil")
	}

	if !client.Timestamp.Equal(clonedClient.Timestamp) {
		t.Errorf("Timestamp field of cloned client is modified")
	}
}
