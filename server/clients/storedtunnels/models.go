package storedtunnels

import "time"

type StoredTunnel struct {
	ID         string    `json:"id" db:"id"`
	ClientID   string    `json:"-" db:"client_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	Name       string    `json:"name" db:"name"`
	Scheme     string    `json:"scheme" db:"scheme"`
	RemoteIP   string    `json:"remote_ip" db:"remote_ip"`
	RemotePort int       `json:"remote_port" db:"remote_port"`
	ACL        string    `json:"acl" db:"acl"`
}
