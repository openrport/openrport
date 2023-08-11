package models

import "time"

type IPAddresses struct {
	IPv4      string    `json:"ipv4"`
	IPv6      string    `json:"ipv6"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error"`
}
