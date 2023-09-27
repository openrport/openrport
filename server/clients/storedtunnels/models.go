package storedtunnels

import (
	"time"

	"github.com/openrport/openrport/share/types"
)

type StoredTunnel struct {
	ID             string            `json:"id" db:"id"`
	ClientID       string            `json:"-" db:"client_id"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	Name           string            `json:"name" db:"name"`
	Scheme         *string           `json:"scheme" db:"scheme"`
	RemoteIP       *string           `json:"remote_ip" db:"remote_ip"`
	RemotePort     *int              `json:"remote_port" db:"remote_port"`
	PublicPort     *int              `json:"public_port" db:"public_port"`
	ACL            *string           `json:"acl" db:"acl"`
	FurtherOptions *types.JSONString `json:"further_options" db:"further_options"`
}
